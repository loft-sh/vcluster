package limitclasses

import (
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Test limitclass on fromHost", ginkgo.Ordered, func() {
	var (
		f             *framework.Framework
		fssdClassName = "fast-ssd"
		fsClassName   = "fast-storage"

		labelValue1 = "one"
		labelValue2 = "two"

		fssdPvc     = "fast-ssd-pvc"
		fstoragePvc = "fast-storage-pvc"

		testNamespace = "default"
		hostNamespace = "vcluster"
	)

	ginkgo.BeforeAll(func() {
		f = framework.DefaultFramework
		// Create fast-ssd storageClass on host
		fastSsdClass := &storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Name:   fssdClassName,
				Labels: map[string]string{"value": labelValue1},
			},
			Provisioner:          "csi.driver.example.com",
			ReclaimPolicy:        func() *corev1.PersistentVolumeReclaimPolicy { rp := corev1.PersistentVolumeReclaimDelete; return &rp }(),
			AllowVolumeExpansion: func() *bool { b := true; return &b }(),
			VolumeBindingMode:    func() *storagev1.VolumeBindingMode { m := storagev1.VolumeBindingWaitForFirstConsumer; return &m }(),
			Parameters:           map[string]string{"type": "ssd"},
		}

		_, err := f.HostClient.StorageV1().StorageClasses().Create(f.Context, fastSsdClass, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		// Create fast-storage storageClass on host
		fastStorageClass := &storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Name:   fsClassName,
				Labels: map[string]string{"value": labelValue2},
			},
			Provisioner:          "csi.driver.example.com",
			ReclaimPolicy:        func() *corev1.PersistentVolumeReclaimPolicy { rp := corev1.PersistentVolumeReclaimDelete; return &rp }(),
			AllowVolumeExpansion: func() *bool { b := true; return &b }(),
			MountOptions:         []string{"discard"},
			VolumeBindingMode:    func() *storagev1.VolumeBindingMode { m := storagev1.VolumeBindingWaitForFirstConsumer; return &m }(),
			Parameters:           map[string]string{"type": "ssd"},
		}

		_, err = f.HostClient.StorageV1().StorageClasses().Create(f.Context, fastStorageClass, metav1.CreateOptions{})
		framework.ExpectNoError(err)

	})

	ginkgo.AfterAll(func() {
		_ = f.HostClient.StorageV1().StorageClasses().Delete(f.Context, fssdClassName, metav1.DeleteOptions{})
		_ = f.HostClient.StorageV1().StorageClasses().Delete(f.Context, fsClassName, metav1.DeleteOptions{})
		_ = f.HostClient.CoreV1().PersistentVolumeClaims(testNamespace).Delete(f.Context, fssdPvc, metav1.DeleteOptions{})
		_ = f.HostClient.CoreV1().PersistentVolumeClaims(testNamespace).Delete(f.Context, fstoragePvc, metav1.DeleteOptions{})
	})

	ginkgo.It("should only sync storageClasses with allowed label to vcluster", func() {
		gomega.Eventually(func() []string {
			scs, err := f.VClusterClient.StorageV1().StorageClasses().List(f.Context, metav1.ListOptions{}) // List all storageClasses in the vCluster
			if err != nil {
				return nil
			}
			var names []string
			for _, sc := range scs.Items {
				names = append(names, sc.Name)
			}
			return names
		}).WithTimeout(time.Minute).WithPolling(time.Second).
			Should(gomega.ContainElement(fssdClassName))

		gomega.Consistently(func() []string {
			scs, err := f.VClusterClient.NetworkingV1().IngressClasses().List(f.Context, metav1.ListOptions{})
			if err != nil {
				return nil
			}
			var names []string
			for _, sc := range scs.Items {
				names = append(names, sc.Name)
			}
			return names
		}).WithTimeout(5 * time.Second).WithPolling(time.Second).
			ShouldNot(gomega.ContainElement(fsClassName))
	})

	ginkgo.It("should not sync vcluster PVCs using a filtered storageClasses to host", func() {
		// Try to create a PVC using fast-storage storageClass in vcluster
		fastStoragepvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fstoragePvc,
				Namespace: testNamespace,
				Labels:    map[string]string{"value": labelValue2},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("10Gi"),
					},
				},
				StorageClassName: func() *string { s := fsClassName; return &s }(),
			},
		}
		_, err := f.VClusterClient.CoreV1().PersistentVolumeClaims(testNamespace).Create(f.Context, fastStoragepvc, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		// It should NOT be synced to host
		time.Sleep(5 * time.Second)
		_, err = f.HostClient.CoreV1().PersistentVolumeClaims(testNamespace).Get(f.Context, fssdClassName, metav1.GetOptions{})
		gomega.Expect(err).To(gomega.HaveOccurred())
	})

	ginkgo.It("should sync vcluster PVCs using allowed storageClass to host", func() {
		// Try to create a PVC using fast-ssd storageClass in vcluster
		fastssdpvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fssdPvc,
				Namespace: testNamespace,
				Labels:    map[string]string{"value": labelValue1},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("5Gi"),
					},
				},
				StorageClassName: func() *string { s := fssdClassName; return &s }(),
			},
		}
		_, err := f.VClusterClient.CoreV1().PersistentVolumeClaims(testNamespace).Create(f.Context, fastssdpvc, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		// It should be synced to host
		gomega.Eventually(func() []string {
			scs, err := f.HostClient.CoreV1().PersistentVolumeClaims(hostNamespace).List(f.Context, metav1.ListOptions{}) // List all PVCs in the vCluster
			if err != nil {
				return nil
			}
			var names []string
			for _, sc := range scs.Items {
				names = append(names, sc.Name)
			}
			return names
		}).WithTimeout(time.Minute).WithPolling(time.Second).
			Should(gomega.ContainElement(fssdPvc + "-x-" + testNamespace + "-x-" + hostNamespace))
	})
})
