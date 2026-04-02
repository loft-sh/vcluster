package certs

import (
	"context"
	"os"
	"time"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	certscmd "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/certs"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DescribeCertExpiration registers tests that verify vCluster can recover from
// an expired CA certificate. The CA is rotated with a 1-second validity period,
// then rotated again once the cert expires.
// Ordered because the specs form a lifecycle: check validity -> issue short-lived CA ->
// wait for expiry -> rotate-ca to recover -> verify new CA is valid.
func DescribeCertExpiration(vcluster suite.Dependency) bool {
	return Describe("vCluster cert expiration recovery",
		Ordered,
		labels.Core,
		labels.Security,
		cluster.Use(vcluster),
		cluster.Use(clusters.HostCluster),
		func() {
			var (
				hostClient          kubernetes.Interface
				vClusterName        string
				vClusterNamespace   string
				caFingerprintBefore string
			)

			BeforeAll(func(ctx context.Context) context.Context {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterNamespace = "vcluster-" + vClusterName
				return ctx
			})

			// Confirm CA cert is currently valid and capture fingerprint for later comparison.
			It("should confirm current CA cert is valid and capture fingerprint", func(ctx context.Context) {
				secret, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
					certs.CertSecretName(vClusterName), metav1.GetOptions{})
				Expect(err).To(Succeed())

				cert, err := parseCertFromPEM(secret.Data[certs.CACertName])
				Expect(err).To(Succeed(), "parsing CA cert")
				Expect(cert.NotAfter.After(time.Now())).To(BeTrue(), "CA cert should be valid before the test")

				caFingerprintBefore = certFingerprint(cert)

				certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vClusterNamespace})
				certsCmd.SetArgs([]string{"check", vClusterName})
				Expect(certsCmd.Execute()).To(Succeed())
			})

			// Issue a CA with 1-second validity to trigger near-immediate expiry.
			It("should rotate CA with a 1-second validity period", func(_ context.Context) {
				DeferCleanup(func(_ context.Context) {
					os.Unsetenv("DEVELOPMENT")
					os.Unsetenv("VCLUSTER_CERTS_VALIDITYPERIOD")
				})
				os.Setenv("DEVELOPMENT", "true")
				os.Setenv("VCLUSTER_CERTS_VALIDITYPERIOD", "1s")

				certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vClusterNamespace})
				certsCmd.SetArgs([]string{"rotate-ca", vClusterName})
				Expect(certsCmd.Execute()).To(Succeed())
			})

			// Wait until vCluster pods are running after the short-lived CA rotation.
			It("should have all pods running after short-lived CA rotation", func(ctx context.Context) {
				waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutLong)
			})

			// Wait for the short-lived CA to actually expire.
			It("should detect that the CA cert has expired", func(ctx context.Context) {
				Eventually(func(g Gomega) {
					secret, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
						certs.CertSecretName(vClusterName), metav1.GetOptions{})
					g.Expect(err).To(Succeed(), "getting cert secret")

					cert, err := parseCertFromPEM(secret.Data[certs.CACertName])
					g.Expect(err).To(Succeed(), "parsing CA cert")

					g.Expect(cert.NotAfter.Before(time.Now())).To(BeTrue(),
						"CA cert not expired yet (expires at %s)", cert.NotAfter)
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
			})

			// Rotate the expired CA to restore normal operation.
			It("should rotate the expired CA cert to recover", func(_ context.Context) {
				certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vClusterNamespace})
				certsCmd.SetArgs([]string{"rotate-ca", vClusterName})
				Expect(certsCmd.Execute()).To(Succeed())
			})

			// Wait for full recovery after rotating the expired CA.
			It("should have all pods ready after rotating the expired CA", func(ctx context.Context) {
				waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutLong)
			})

			// Confirm certs check reports the new CA and it is valid.
			It("should report new cert expiry via vcluster certs check after recovery", func(_ context.Context) {
				certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vClusterNamespace})
				certsCmd.SetArgs([]string{"check", vClusterName})
				Expect(certsCmd.Execute()).To(Succeed())
			})

			// Confirm new CA cert is valid, has changed fingerprint, and has normal validity.
			It("should confirm new CA cert is renewed after expiration recovery", func(ctx context.Context) {
				secret, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
					certs.CertSecretName(vClusterName), metav1.GetOptions{})
				Expect(err).To(Succeed())

				cert, err := parseCertFromPEM(secret.Data[certs.CACertName])
				Expect(err).To(Succeed(), "parsing recovered CA cert")

				Expect(certFingerprint(cert)).NotTo(Equal(caFingerprintBefore),
					"new CA fingerprint should differ from the short-lived CA")
				Expect(cert.NotAfter.After(time.Now().Add(24*time.Hour))).To(BeTrue(),
					"new CA cert should have normal validity (>24h) after recovery")
			})

			// Reconnect after all rotations.
			It("should reconnect to vCluster after expiration recovery", func(ctx context.Context) {
				reconnectVCluster(ctx, vClusterName, vClusterNamespace)
			})
		},
	)
}
