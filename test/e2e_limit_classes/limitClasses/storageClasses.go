package limitclasses

import (
	"fmt"
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
		scs, err := f.VClusterClient.StorageV1().StorageClasses().List(f.Context, metav1.ListOptions{}) // List all storageClasses in the vCluster
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		var names []string
		for _, sc := range scs.Items {
			names = append(names, sc.Name)
		}
		gomega.Expect(names).To(gomega.ContainElement(fssdClassName))
		ginkgo.By("Found fast-ssd in vcluster")
		gomega.Expect(names).NotTo(gomega.ContainElement(fsClassName))
		ginkgo.By("fast-storage is not available in vcluster")
	})

	ginkgo.It("should not sync vcluster PVCs using a filtered storageClasses to host", func() {
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
		// eventList, err := f.VClusterClient.CoreV1().Events(testNamespace).List(f.Context, metav1.ListOptions{
		// 	FieldSelector: fmt.Sprintf("involvedObject.kind=PersistentVolumeClaim,involvedObject.name=%s", fstoragePvc),
		// })
		// gomega.Expect(err).NotTo(gomega.HaveOccurred())
		// var found bool
		// for _, event := range eventList.Items {
		// 	if event.Type == corev1.EventTypeWarning && event.Reason == "SyncWarning" {
		// 		found = true
		// 		expectedSubstring := fmt.Sprintf(`did not sync persistent volume claim "%s" to host because the storage class "%s" in the host does not match the selector under 'sync.fromHost.storageClasses.selector'`, fstoragePvc, fsClassName)
		// 		gomega.Expect(event.Message).To(gomega.ContainSubstring(expectedSubstring))
		// 		break
		// 	}
		// }
		// gomega.Expect(found).To(gomega.BeTrue(), "Expected to find a SyncWarning event for the ingress with unavailable ingressClass")

		gomega.Eventually(func() bool {
			eventList, err := f.VClusterClient.CoreV1().Events(testNamespace).List(f.Context, metav1.ListOptions{
				FieldSelector: fmt.Sprintf("involvedObject.kind=PersistentVolumeClaim,involvedObject.name=%s", fstoragePvc),
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			for _, event := range eventList.Items {
				if event.Type == corev1.EventTypeWarning && event.Reason == "SyncWarning" {
					expectedSubstring := fmt.Sprintf(`did not sync persistent volume claim "%s" to host because the storage class "%s" in the host does not match the selector under 'sync.fromHost.storageClasses.selector'`, fstoragePvc, fsClassName)
					gomega.Expect(event.Message).To(gomega.ContainSubstring(expectedSubstring))
					return true
				}
			}
			return false
		}).WithTimeout(time.Minute).WithPolling(time.Second).Should(gomega.BeTrue(), "Timed out waiting for SyncWarning event for PVC %s", fstoragePvc)
	})

	ginkgo.It("should sync vcluster PVCs using allowed storageClass to host", func() {
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
		_, err := f.VClusterClient.CoreV1().PersistentVolumeClaims(testNamespace).Create(f.Context, fastssdpvc, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		ginkgo.By("PVC should be synced to host")
		gomega.Eventually(func() []string {
			scs, err := f.HostClient.CoreV1().PersistentVolumeClaims(hostNamespace).List(f.Context, metav1.ListOptions{})
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
