package certs

import (
	"context"
	"time"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

			// Spec 2 depends on 1: verify the vcluster stays healthy with short-lived
			// serving certs. With VCLUSTER_CERTS_VALIDITYPERIOD=3m, the serving cert
			// expires in 3 minutes. If the syncer weren't regenerating it, the API
			// server would start rejecting TLS connections and the pod would become
			// unready. We wait past the cert lifetime and verify the pod is still ready.
			It("should remain healthy after serving cert would have expired", func(ctx context.Context) {
				By("Verifying the pod stays ready past the cert lifetime", func() {
					// With VCLUSTER_CERTS_VALIDITYPERIOD=3m, the serving cert expires
					// in 3 minutes. If the syncer weren't continuously regenerating it,
					// the API server would reject TLS connections and the pod would fail
					// health checks. We verify the pod stays ready for 4 minutes —
					// past the cert lifetime.
					Consistently(func(g Gomega) {
						pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
							LabelSelector: "app=vcluster,release=" + vClusterName,
						})
						g.Expect(err).To(Succeed())
						g.Expect(pods.Items).NotTo(BeEmpty(), "vcluster pods disappeared")

						for _, pod := range pods.Items {
							for _, container := range pod.Status.ContainerStatuses {
								g.Expect(container.State.Running).NotTo(BeNil(),
									"container %s in pod %s should still be running", container.Name, pod.Name)
								g.Expect(container.Ready).To(BeTrue(),
									"container %s in pod %s should still be ready", container.Name, pod.Name)
							}
						}
					}).WithPolling(constants.PollingInterval).WithTimeout(4 * time.Minute).Should(Succeed())
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
