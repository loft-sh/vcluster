package fromhost

import (
	"context"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DescribeFromHostStorageClasses registers storageClass sync from host tests against the given vCluster.
func DescribeFromHostStorageClasses(vcluster suite.Dependency) bool {
	return Describe("StorageClasses sync from host",
		labels.Core,
		labels.Sync,
		labels.StorageClasses,
		cluster.Use(vcluster),
		func() {
			var (
				hostClient     kubernetes.Interface
				vClusterClient kubernetes.Interface
				vClusterName   string
				vClusterHostNS string
			)

			BeforeEach(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterHostNS = "vcluster-" + vClusterName
			})

			// createStorageClass creates a StorageClass on the host and registers cleanup.
			createStorageClass := func(ctx context.Context, name string, scLabels map[string]string, mountOptions []string) *storagev1.StorageClass {
				GinkgoHelper()
				reclaimPolicy := corev1.PersistentVolumeReclaimDelete
				allowExpansion := true
				bindingMode := storagev1.VolumeBindingWaitForFirstConsumer
				sc := &storagev1.StorageClass{
					ObjectMeta: metav1.ObjectMeta{
						Name:   name,
						Labels: scLabels,
					},
					Provisioner:          "csi.driver.example.com",
					ReclaimPolicy:        &reclaimPolicy,
					AllowVolumeExpansion: &allowExpansion,
					MountOptions:         mountOptions,
					VolumeBindingMode:    &bindingMode,
					Parameters:           map[string]string{"type": "ssd"},
				}
				created, err := hostClient.StorageV1().StorageClasses().Create(ctx, sc, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					err := hostClient.StorageV1().StorageClasses().Delete(ctx, name, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})
				return created
			}

			It("only syncs storageClasses matching the label selector to vcluster", func(ctx context.Context) {
				suffix := fmt.Sprintf("%d", GinkgoRandomSeed())
				matchingName := "sc-match-" + suffix
				nonMatchingName := "sc-nomatch-" + suffix

				createStorageClass(ctx, matchingName, map[string]string{"value": "one"}, nil)
				createStorageClass(ctx, nonMatchingName, map[string]string{"value": "two"}, []string{"discard"})

				By("waiting for the matching class to appear and the non-matching class to stay absent", func() {
					Eventually(func(g Gomega) {
						storageClasses, err := vClusterClient.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
						g.Expect(err).To(Succeed(), "failed to list storageClasses in vcluster: %v", err)

						var foundMatch, foundNoMatch bool
						for _, sc := range storageClasses.Items {
							switch sc.Name {
							case matchingName:
								foundMatch = true
							case nonMatchingName:
								foundNoMatch = true
							}
						}
						g.Expect(foundMatch).To(BeTrue(), "expected matching storageClass to be synced to vcluster")
						g.Expect(foundNoMatch).To(BeFalse(), "expected non-matching storageClass to stay absent from vcluster")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("does not sync PVCs created in vcluster using a storageClass not available in vcluster", func(ctx context.Context) {
				suffix := fmt.Sprintf("%d", GinkgoRandomSeed())
				nonMatchingName := "sc-pvcreject-" + suffix
				pvcName := "pvc-reject-" + suffix

				createStorageClass(ctx, nonMatchingName, map[string]string{"value": "two"}, []string{"discard"})

				By("creating a PVC using the non-synced storageClass in vcluster", func() {
					_, err := vClusterClient.CoreV1().PersistentVolumeClaims("default").Create(ctx, &corev1.PersistentVolumeClaim{
						ObjectMeta: metav1.ObjectMeta{
							Name:      pvcName,
							Namespace: "default",
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("10Gi"),
								},
							},
							StorageClassName: func() *string { s := nonMatchingName; return &s }(),
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := vClusterClient.CoreV1().PersistentVolumeClaims("default").Delete(ctx, pvcName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).To(Succeed())
						}
					})
				})

				By("verifying the PVC is not synced to the host", func() {
					translatedName := translate.SafeConcatName(pvcName, "x", "default", "x", vClusterName)
					_, err := hostClient.CoreV1().PersistentVolumeClaims(vClusterHostNS).Get(ctx, translatedName, metav1.GetOptions{})
					Expect(kerrors.IsNotFound(err)).To(BeTrue(), "PVC using non-synced storageClass should not appear on host")
				})

				By("waiting for a SyncWarning event on the PVC", func() {
					expectedMsg := fmt.Sprintf(
						`did not sync persistent volume claim "%s" to host because the storage class "%s" in the host does not match the selector under 'sync.fromHost.storageClasses.selector'`,
						pvcName, nonMatchingName,
					)
					Eventually(func(g Gomega) {
						eventList, err := vClusterClient.CoreV1().Events("default").List(ctx, metav1.ListOptions{})
						g.Expect(err).To(Succeed(), "failed to list events: %v", err)
						var found bool
						for _, event := range eventList.Items {
							if event.InvolvedObject.Kind == "PersistentVolumeClaim" &&
								event.InvolvedObject.Name == pvcName &&
								event.Type == corev1.EventTypeWarning &&
								event.Reason == "SyncWarning" {
								g.Expect(event.Message).To(ContainSubstring(expectedMsg))
								found = true
								break
							}
						}
						g.Expect(found).To(BeTrue(), "expected SyncWarning event for PVC %s", pvcName)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})

			It("syncs PVCs created in vcluster to host when using a storageClass synced from host", func(ctx context.Context) {
				suffix := fmt.Sprintf("%d", GinkgoRandomSeed())
				matchingName := "sc-pvcsync-" + suffix
				pvcName := "pvc-sync-" + suffix

				createStorageClass(ctx, matchingName, map[string]string{"value": "one"}, nil)

				By("waiting for the storageClass to be synced to vcluster", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterClient.StorageV1().StorageClasses().Get(ctx, matchingName, metav1.GetOptions{})
						g.Expect(err).To(Succeed(), "storageClass %s not yet synced to vcluster: %v", matchingName, err)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				By("creating a PVC using the synced storageClass in vcluster", func() {
					_, err := vClusterClient.CoreV1().PersistentVolumeClaims("default").Create(ctx, &corev1.PersistentVolumeClaim{
						ObjectMeta: metav1.ObjectMeta{
							Name:      pvcName,
							Namespace: "default",
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("5Gi"),
								},
							},
							StorageClassName: func() *string { s := matchingName; return &s }(),
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := vClusterClient.CoreV1().PersistentVolumeClaims("default").Delete(ctx, pvcName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).To(Succeed())
						}
					})
				})

				By("waiting for the PVC to appear in the host vcluster namespace", func() {
					expectedHostPVCName := translate.SafeConcatName(pvcName, "x", "default", "x", vClusterName)
					Eventually(func(g Gomega) {
						pvcs, err := hostClient.CoreV1().PersistentVolumeClaims(vClusterHostNS).List(ctx, metav1.ListOptions{})
						g.Expect(err).To(Succeed(), "failed to list PVCs in host namespace %s: %v", vClusterHostNS, err)
						var found bool
						for _, pvc := range pvcs.Items {
							if pvc.Name == expectedHostPVCName {
								found = true
								break
							}
						}
						g.Expect(found).To(BeTrue(), "expected PVC %s to appear in host namespace %s", expectedHostPVCName, vClusterHostNS)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})
		})
}
