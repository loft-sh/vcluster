package certs

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"time"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DescribeServingCertRotation verifies that the API server serving certificate
// syncer correctly hot-reloads the cert at runtime when it approaches expiry.
// This test requires a vcluster deployed with DEVELOPMENT=true and
// VCLUSTER_CERTS_VALIDITYPERIOD=3m so the serving cert has a short lifetime.
//
// The syncer polls every 2 seconds and regenerates the cert when IsCertExpired
// returns true (<=90 days to expiry). With a 3-minute cert, this fires almost
// immediately, verifying the fix for the SANs bug where expiry-triggered
// regeneration was silently discarded.
func DescribeServingCertRotation(vcluster suite.Dependency) bool {
	return Describe("API server serving cert runtime rotation",
		Ordered,
		labels.Core,
		labels.Security,
		cluster.Use(vcluster),
		cluster.Use(clusters.HostCluster),
		func() {
			var (
				hostClient        kubernetes.Interface
				vClusterName      string
				vClusterNamespace string
			)

			BeforeAll(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterNamespace = "vcluster-" + vClusterName
			})

			// Spec 1: vcluster is ready
			It("should have the vCluster pod running and ready", func(ctx context.Context) {
				Eventually(func(g Gomega) {
					pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
						LabelSelector: "app=vcluster,release=" + vClusterName,
					})
					g.Expect(err).To(Succeed())
					g.Expect(pods.Items).NotTo(BeEmpty(), "no vcluster pods found")
					for _, pod := range pods.Items {
						for _, container := range pod.Status.ContainerStatuses {
							g.Expect(container.State.Running).NotTo(BeNil(),
								"container %s in pod %s should be running", container.Name, pod.Name)
							g.Expect(container.Ready).To(BeTrue(),
								"container %s in pod %s should be ready", container.Name, pod.Name)
						}
					}
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
			})

			// Spec 2 depends on 1: record the initial serving cert serial number
			var initialSerial string

			It("should have a short-lived serving cert", func(ctx context.Context) {
				By("Getting the vcluster service endpoint", func() {
					svc, err := hostClient.CoreV1().Services(vClusterNamespace).Get(ctx,
						vClusterName, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(svc.Spec.ClusterIP).NotTo(BeEmpty())
				})

				By("Connecting via TLS and recording the initial cert", func() {
					servingCert := getServingCert(ctx, hostClient, vClusterName, vClusterNamespace)
					Expect(servingCert).NotTo(BeNil(), "should get a serving cert via TLS")

					initialSerial = servingCert.SerialNumber.String()
					Expect(initialSerial).NotTo(BeEmpty())

					// With VCLUSTER_CERTS_VALIDITYPERIOD=3m, the cert should expire
					// within minutes, not a year.
					Expect(servingCert.NotAfter.Before(time.Now().Add(10 * time.Minute))).To(BeTrue(),
						"serving cert should be short-lived (expires %s), not 365 days",
						servingCert.NotAfter.Format(time.RFC3339))
				})
			})

			// Spec 3 depends on 2: wait for the syncer to regenerate the cert
			// The syncer runs every 2 seconds and checks IsCertExpired (90-day threshold).
			// With a 3-minute cert, the cert is always within the 90-day window,
			// so the syncer should regenerate it on the next poll cycle.
			It("should hot-reload the serving cert without pod restart", func(ctx context.Context) {
				By("Waiting for the serving cert serial to change", func() {
					Eventually(func(g Gomega) {
						servingCert := getServingCert(ctx, hostClient, vClusterName, vClusterNamespace)
						g.Expect(servingCert).NotTo(BeNil(), "should get a serving cert")

						newSerial := servingCert.SerialNumber.String()
						g.Expect(newSerial).NotTo(Equal(initialSerial),
							"serving cert serial should change after syncer regeneration")
					}).WithPolling(2 * time.Second).WithTimeout(30 * time.Second).Should(Succeed())
				})

				By("Verifying the pod was NOT restarted", func() {
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

// getServingCert connects to the vcluster's API server via port-forward and
// returns the serving certificate from the TLS handshake. Returns nil if the
// connection fails.
func getServingCert(ctx context.Context, hostClient kubernetes.Interface, vClusterName, vClusterNamespace string) *x509.Certificate {
	// Get the service ClusterIP to connect to the vcluster API server
	svc, err := hostClient.CoreV1().Services(vClusterNamespace).Get(ctx,
		vClusterName, metav1.GetOptions{})
	if err != nil {
		return nil
	}

	// The vcluster service exposes the API server on port 443
	addr := fmt.Sprintf("%s:443", svc.Spec.ClusterIP)
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 5 * time.Second},
		"tcp",
		addr,
		&tls.Config{InsecureSkipVerify: true},
	)
	if err != nil {
		return nil
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil
	}
	return certs[0]
}
