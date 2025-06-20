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
		ginkgo.By("Creating fast-ssd storageClass on host")
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

		ginkgo.By("Creating fast-storage storageClass on host")
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
		ginkgo.By("Listing all storageClasses in vCluster")
		gomega.Eventually(func() bool {
			storageClasses, err := f.VClusterClient.StorageV1().StorageClasses().List(f.Context, metav1.ListOptions{})
			if err != nil {
				return false
			}
			for _, storageClass := range storageClasses.Items {
				if storageClass.Name == fssdClassName {
					return true
				}
			}
			return false
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(gomega.BeTrue(), "Timed out waiting for listing all storageClasses")

		gomega.Consistently(func() bool {
			storageClasses, err := f.VClusterClient.StorageV1().StorageClasses().List(f.Context, metav1.ListOptions{})
			if err != nil {
				return false
			}
			for _, storageClass := range storageClasses.Items {
				if storageClass.Name == fsClassName {
					return true
				}
			}
			return false
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(gomega.BeFalse(), "Timed out waiting for listing all storageClasses")
	})

	ginkgo.It("should not sync vcluster PVCs created using an storageClass not available in vCluster", func() {
		ginkgo.By("Creating a PVC using fast-storage storageClass in vcluster")
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

		ginkgo.By("PVC should not be synced to host")
		_, err = f.HostClient.CoreV1().PersistentVolumeClaims(testNamespace).Get(f.Context, fssdClassName, metav1.GetOptions{})
		gomega.Expect(err).To(gomega.HaveOccurred())

		ginkgo.By("There should be a warning message event in the describe of the created PVC")
		gomega.Eventually(func() bool {
      eventList, err := f.VClusterClient.CoreV1().Events(testNamespace).List(f.Context, metav1.ListOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			for _, event := range eventList.Items {
				if event.InvolvedObject.Kind == "PersistentVolumeClaim" && event.InvolvedObject.Name == fstoragePvc && event.Type == corev1.EventTypeWarning && event.Reason == "SyncWarning" {
					gomega.Expect(event.Message).To(gomega.ContainSubstring(`did not sync persistent volume claim "%s" to host because the storage class "%s" in the host does not match the selector under 'sync.fromHost.storageClasses.selector'`, fstoragePvc, fsClassName))
					return true
				}
			}
			return false
		}).
			WithTimeout(time.Minute).
			WithPolling(time.Second).
			Should(gomega.BeTrue(), "Timed out waiting for SyncWarning event for PVC %s", fstoragePvc)
	})

	ginkgo.It("should sync PVC created in vCluster to host using storageClass synced from Host", func() {
		ginkgo.By("Creating a PVC using fast-ssd storageClass in vcluster")
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
		ginkgo.By("Listing all PVCs in host's vcluster namespace")
		_, err := f.VClusterClient.CoreV1().PersistentVolumeClaims(testNamespace).Create(f.Context, fastssdpvc, metav1.CreateOptions{})
		framework.ExpectNoError(err)
		gomega.Eventually(func() bool {
			storageClasses, err := f.HostClient.CoreV1().PersistentVolumeClaims(hostNamespace).List(f.Context, metav1.ListOptions{})
			if err != nil {
				return false
			}
			for _, pvc := range storageClasses.Items {
				if pvc.Name == fssdPvc+"-x-"+testNamespace+"-x-"+hostNamespace {
					return true
				}
			}
			return false
		}).
			WithTimeout(time.Minute).
			WithPolling(time.Second).
			Should(gomega.BeTrue(), "Timed out waiting for listing all PVCs in host")
	})
})
