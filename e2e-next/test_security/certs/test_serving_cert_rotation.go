package certs

import (
	"context"
	"time"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ServingCertRotationSpec verifies that the API server serving certificate
// syncer correctly regenerates the cert at runtime when it approaches expiry.
// This test requires a vcluster deployed with DEVELOPMENT=true and
// VCLUSTER_CERTS_VALIDITYPERIOD=3m so the serving cert has a short lifetime.
//
// The syncer polls every 2 seconds and regenerates the cert when IsCertExpired
// returns true (<=90 days to expiry). With a 3-minute cert, this fires on every
// poll cycle. We verify regeneration by recording the cert serial via TLS
// handshake, waiting briefly, then verifying the serial changed — proving
// the syncer actually regenerated and applied a new serving cert.
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
				hostRestConfig    *rest.Config
				vClusterName      string
				vClusterNamespace string
				initialSerial     string
				baselineRestarts  map[string]int32
			)

			BeforeAll(func(ctx context.Context) context.Context {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				hostRestConfig = cluster.From(ctx, constants.GetHostClusterName()).KubernetesRestConfig()
				Expect(hostRestConfig).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterNamespace = "vcluster-" + vClusterName
				return ctx
			})

			It("should have the vCluster pod running and ready", func(ctx context.Context) {
				By("Waiting for the pod to be ready", func() {
					waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutVeryLong)
				})

				By("Recording baseline restart counts and initial serving cert serial", func() {
					selector := "app=vcluster,release=" + vClusterName
					baselineRestarts = containerRestartCounts(ctx, hostClient, vClusterNamespace, selector)

					pods := listPodsBySelector(ctx, hostClient, vClusterNamespace, selector)
					Expect(pods).NotTo(BeEmpty())

					var err error
					initialSerial, err = getServingCertSerial(ctx, hostRestConfig, hostClient, vClusterNamespace, pods[0].Name)
					Expect(err).To(Succeed(), "failed to get initial serving cert serial")
					Expect(initialSerial).NotTo(BeEmpty())
				})
			})

			// Spec 2 depends on 1: wait briefly then verify the serving cert serial
			// changed. With VCLUSTER_CERTS_VALIDITYPERIOD=3m, the cert is always
			// within the 90-day expiry window, so the syncer regenerates it every
			// 2 seconds. A changed serial proves the cert was actually regenerated.
			It("should regenerate the serving cert with a new serial", func(ctx context.Context) {
				By("Waiting for the serving cert serial to change", func() {
					pods := listPodsBySelector(ctx, hostClient, vClusterNamespace, "app=vcluster,release="+vClusterName)
					Expect(pods).NotTo(BeEmpty())

					Eventually(func(g Gomega) {
						serial, err := getServingCertSerial(ctx, hostRestConfig, hostClient, vClusterNamespace, pods[0].Name)
						g.Expect(err).To(Succeed(), "failed to get serving cert serial")
						g.Expect(serial).NotTo(Equal(initialSerial),
							"serving cert serial should have changed after syncer regeneration")
					}).WithPolling(5 * time.Second).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			// Spec 3 depends on 2: verify the pod was NOT restarted — serving cert
			// rotation should be a hot-reload, not a restart.
			It("should not have restarted the pod for serving cert rotation", func(ctx context.Context) {
				By("Verifying no container restart counts increased", func() {
					selector := "app=vcluster,release=" + vClusterName
					currentRestarts := containerRestartCounts(ctx, hostClient, vClusterNamespace, selector)

					for key, baseline := range baselineRestarts {
						current, ok := currentRestarts[key]
						Expect(ok).To(BeTrue(), "container %s disappeared after cert rotation", key)
						Expect(current).To(BeNumerically("==", baseline),
							"container %s restarted during serving cert rotation (was %d, now %d)",
							key, baseline, current)
					}
				})
			})
		},
	)
}
