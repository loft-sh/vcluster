package certs

import (
	"context"
	"strings"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ServingCertRotationSpec verifies that the API server serving certificate
// syncer correctly regenerates the cert at runtime when it approaches expiry.
// This test requires a vcluster deployed with DEVELOPMENT=true and
// VCLUSTER_CERTS_VALIDITYPERIOD=3m so the serving cert has a short lifetime.
//
// The syncer polls every 2 seconds and regenerates the cert when IsCertExpired
// returns true (<=90 days to expiry). With a 3-minute cert, this fires on every
// poll cycle. We verify regeneration by checking the syncer logs for the
// "Generated serving cert for sans" message, which is emitted each time the
// cert is regenerated and applied.
//
// This validates the fix for the SANs bug where expiry-triggered regeneration
// was silently discarded because the syncer compared SAN lists instead of cert bytes.
//
// Must be called inside a Describe that has cluster.Use() for the vcluster and host cluster.
func ServingCertRotationSpec() {
	Describe("API server serving cert runtime rotation",
		Ordered,
		labels.Core,
		labels.Security,
		func() {
			var (
				hostClient        kubernetes.Interface
				vClusterName      string
				vClusterNamespace string
			)

			BeforeAll(func(ctx context.Context) context.Context {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterNamespace = "vcluster-" + vClusterName
				return ctx
			})

			It("should have the vCluster pod running and ready", func(ctx context.Context) {
				waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutVeryLong)
			})

			// Spec 2 depends on 1: verify the syncer is regenerating the serving cert.
			// With VCLUSTER_CERTS_VALIDITYPERIOD=3m, the cert is always within the
			// 90-day expiry window, so the syncer should regenerate it every 2 seconds.
			// We verify by checking the syncer container logs for the regeneration message.
			It("should continuously regenerate the serving cert due to short lifetime", func(ctx context.Context) {
				By("Checking syncer logs for repeated cert regeneration entries", func() {
					Eventually(func(g Gomega) {
						pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
							LabelSelector: "app=vcluster,release=" + vClusterName,
						})
						g.Expect(err).To(Succeed())
						g.Expect(pods.Items).NotTo(BeEmpty())

						pod := pods.Items[0]
						logs, err := hostClient.CoreV1().Pods(vClusterNamespace).GetLogs(
							pod.Name, &corev1.PodLogOptions{
								Container:    "syncer",
								SinceSeconds: int64Ptr(30),
							}).DoRaw(ctx)
						g.Expect(err).To(Succeed())

						// The syncer logs "Generated serving cert for sans: ..." each time
						// it regenerates and applies the cert. With the SANs bug, this
						// message only appeared on the first generation (SAN change from nil),
						// never again. With the fix, it appears on every expiry-triggered
						// regeneration.
						regenCount := strings.Count(string(logs), "Generated serving cert for sans")
						g.Expect(regenCount).To(BeNumerically(">=", 2),
							"syncer should have regenerated the serving cert multiple times in the last 30s, "+
								"but found %d regeneration log entries. This indicates the SANs bug is still present — "+
								"expiry-triggered cert regeneration is being silently discarded.", regenCount)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			// Spec 3 depends on 2: verify the pod was NOT restarted — serving cert
			// rotation should be a hot-reload, not a restart.
			It("should not have restarted the pod for serving cert rotation", func(ctx context.Context) {
				By("Verifying all container restart counts are zero", func() {
					pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
						LabelSelector: "app=vcluster,release=" + vClusterName,
					})
					Expect(err).To(Succeed())
					Expect(pods.Items).NotTo(BeEmpty())

					for _, pod := range pods.Items {
						for _, container := range pod.Status.ContainerStatuses {
							Expect(container.RestartCount).To(BeNumerically("==", 0),
								"container %s in pod %s should not have restarted (serving cert hot-reload should not require restart)",
								container.Name, pod.Name)
						}
					}
				})
			})
		},
	)
}

func int64Ptr(i int64) *int64 { return &i }
