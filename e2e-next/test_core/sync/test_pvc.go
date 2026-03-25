package test_core

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DescribePVCSync registers PVC sync tests against the given vCluster.
func DescribePVCSync(vcluster suite.Dependency) bool {
	return Describe("PVC sync from vCluster to host",
		labels.Core,
		labels.Sync,
		labels.PVCs,
		labels.Storage,
		cluster.Use(vcluster),
		cluster.Use(clusters.HostCluster),
		func() {
			var (
				hostClient     kubernetes.Interface
				vClusterClient kubernetes.Interface
				vClusterName   string
			)

			BeforeEach(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
			})

			It("should bind a PVC to a dynamically provisioned PV, sync it back, and clean up properly", func(ctx context.Context) {
				suffix := random.String(6)
				nsName := "pvc-sync-test-" + suffix
				pvcName := "pvc-" + suffix
				podName := "nginx-pvc-" + suffix
				hostNS := "vcluster-" + vClusterName

				By("Creating a test namespace", func() {
					_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name:   nsName,
							Labels: map[string]string{"testing-ns-label": "testing-ns-label-value"},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				q := resource.MustParse("3Gi")

				_, err := vClusterClient.CoreV1().PersistentVolumeClaims(nsName).Create(ctx, &corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{Name: pvcName},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{corev1.ResourceStorage: q},
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().PersistentVolumeClaims(nsName).Delete(ctx, pvcName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				By("Waiting for the default service account", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterClient.CoreV1().ServiceAccounts(nsName).Get(ctx, "default", metav1.GetOptions{})
						g.Expect(err).To(Succeed())
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				_, err = vClusterClient.CoreV1().Pods(nsName).Create(ctx, &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: podName},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "nginx", Image: "nginx"},
						},
						Volumes: []corev1.Volume{
							{
								Name: "nginx-pvc",
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: pvcName,
									},
								},
							},
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Pods(nsName).Delete(ctx, podName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				By("Waiting for the PVC to become Bound", func() {
					Eventually(func(g Gomega) {
						vpvc, err := vClusterClient.CoreV1().PersistentVolumeClaims(nsName).Get(ctx, pvcName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						g.Expect(vpvc.Status.Phase).To(Equal(corev1.ClaimBound),
							"PVC phase is %s, not yet Bound", vpvc.Status.Phase)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
				})

				By("Verifying PVC status matches between vCluster and host", func() {
					vpvc, err := vClusterClient.CoreV1().PersistentVolumeClaims(nsName).Get(ctx, pvcName, metav1.GetOptions{})
					Expect(err).To(Succeed())

					hostPvcName := translate.SingleNamespaceHostName(pvcName, nsName, vClusterName)
					pvc, err := hostClient.CoreV1().PersistentVolumeClaims(hostNS).Get(ctx, hostPvcName, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vpvc.Status).To(Equal(pvc.Status))
				})

				// Read the PV name before deleting the PVC
				vpvc, err := vClusterClient.CoreV1().PersistentVolumeClaims(nsName).Get(ctx, pvcName, metav1.GetOptions{})
				Expect(err).To(Succeed())
				pvName := vpvc.Spec.VolumeName

				By("Deleting the PVC and pod to test cleanup sync", func() {
					err := vClusterClient.CoreV1().PersistentVolumeClaims(nsName).Delete(ctx, pvcName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
					err = vClusterClient.CoreV1().Pods(nsName).Delete(ctx, podName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				By("Waiting for the PVC to be fully deleted from vCluster", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterClient.CoreV1().PersistentVolumeClaims(nsName).Get(ctx, pvcName, metav1.GetOptions{})
						g.Expect(kerrors.IsNotFound(err)).To(BeTrue(),
							"PVC %s/%s not yet deleted", nsName, pvcName)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Waiting for the PV to be deleted from vCluster", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterClient.CoreV1().PersistentVolumes().Get(ctx, pvName, metav1.GetOptions{})
						g.Expect(kerrors.IsNotFound(err)).To(BeTrue(),
							"PV %s not yet deleted", pvName)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Verifying the host PVC is also deleted", func() {
					hostPvcName := translate.SingleNamespaceHostName(pvcName, nsName, vClusterName)
					Eventually(func(g Gomega) {
						_, err := hostClient.CoreV1().PersistentVolumeClaims(hostNS).Get(ctx, hostPvcName, metav1.GetOptions{})
						g.Expect(kerrors.IsNotFound(err)).To(BeTrue(),
							"host PVC %s/%s not yet deleted", hostNS, hostPvcName)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})
		},
	)
}
