package certs

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
	"time"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	certscmd "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/certs"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// CertTestsSpec registers all cert rotation, expiration, and kubeconfig TLS tests
// in a single Ordered Describe. They MUST run sequentially because:
//  1. All three operate on the same vCluster and do destructive cert rotations
//  2. CertExpiration uses os.Setenv(VCLUSTER_CERTS_VALIDITYPERIOD) which is
//     process-global - running in parallel would poison other cert operations
//  3. Each section's reconnect establishes the proxy for the next section
//
// Lifecycle: rotation (leaf -> CA with fingerprint verification) ->
// expiration (1s CA -> wait expire -> recover) ->
// kubeconfig TLS (baseline -> leaf rotate -> CA rotate -> verify old TLS fails)
func CertTestsSpec() {
	Describe("vCluster cert tests",
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

			// ------------------------------------------------------------------
			// Section 1: Cert rotation with fingerprint verification
			// Lifecycle: pods ready -> capture fingerprints -> rotate leaf ->
			// verify CA unchanged, leaf changed -> rotate CA -> verify both changed
			// ------------------------------------------------------------------
			Context("cert rotation", func() {
				var (
					apiserverCertBefore        *x509.Certificate
					apiserverFingerprintBefore string
					caCertBefore               *x509.Certificate
					caFingerprintBefore        string
				)

				It("should have all vCluster pods running and ready", func(ctx context.Context) {
					waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutVeryLong)
				})

				It("should have the cert secret", func(ctx context.Context) {
					_, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
						certs.CertSecretName(vClusterName), metav1.GetOptions{})
					Expect(err).To(Succeed())
				})

				It("should report cert expiry via vcluster certs check", func(_ context.Context) {
					certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vClusterNamespace})
					certsCmd.SetArgs([]string{"check", vClusterName})
					Expect(certsCmd.Execute()).To(Succeed())
				})

				It("should capture cert fingerprints before rotation", func(ctx context.Context) {
					secret, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
						certs.CertSecretName(vClusterName), metav1.GetOptions{})
					Expect(err).To(Succeed())

					apiserverCertBefore, err = parseCertFromPEM(secret.Data[certs.APIServerCertName])
					Expect(err).To(Succeed(), "parsing apiserver cert")
					apiserverFingerprintBefore = certFingerprint(apiserverCertBefore)

					caCertBefore, err = parseCertFromPEM(secret.Data[certs.CACertName])
					Expect(err).To(Succeed(), "parsing CA cert")
					caFingerprintBefore = certFingerprint(caCertBefore)
				})

				It("should rotate the leaf certs", func(_ context.Context) {
					certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vClusterNamespace})
					certsCmd.SetArgs([]string{"rotate", vClusterName})
					Expect(certsCmd.Execute()).To(Succeed())
				})

				It("should have all pods ready after leaf cert rotation", func(ctx context.Context) {
					waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutVeryLong)
				})

				It("should verify CA unchanged and leaf renewed after leaf rotation", func(ctx context.Context) {
					secret, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
						certs.CertSecretName(vClusterName), metav1.GetOptions{})
					Expect(err).To(Succeed())

					By("Verifying CA cert fingerprint is unchanged after leaf rotation", func() {
						caCertAfter, err := parseCertFromPEM(secret.Data[certs.CACertName])
						Expect(err).To(Succeed(), "parsing CA cert after leaf rotation")
						Expect(certFingerprint(caCertAfter)).To(Equal(caFingerprintBefore),
							"CA fingerprint should not change after leaf-only rotation")
						Expect(caCertAfter.NotAfter).To(Equal(caCertBefore.NotAfter),
							"CA expiry should not change after leaf-only rotation")
						caFingerprintBefore = certFingerprint(caCertAfter)
					})

					By("Verifying apiserver cert fingerprint changed after leaf rotation", func() {
						apiserverCertAfter, err := parseCertFromPEM(secret.Data[certs.APIServerCertName])
						Expect(err).To(Succeed(), "parsing apiserver cert after leaf rotation")
						Expect(certFingerprint(apiserverCertAfter)).NotTo(Equal(apiserverFingerprintBefore),
							"apiserver fingerprint should change after leaf rotation")
						Expect(apiserverCertAfter.NotAfter.After(apiserverCertBefore.NotAfter)).To(BeTrue(),
							"apiserver cert should expire later after leaf rotation")
						apiserverFingerprintBefore = certFingerprint(apiserverCertAfter)
					})
				})

				It("should rotate the CA cert", func(_ context.Context) {
					certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vClusterNamespace})
					certsCmd.SetArgs([]string{"rotate-ca", vClusterName})
					Expect(certsCmd.Execute()).To(Succeed())
				})

				It("should have all pods ready after CA rotation", func(ctx context.Context) {
					waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutVeryLong)
				})

				It("should verify both CA and leaf renewed after CA rotation", func(ctx context.Context) {
					secret, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
						certs.CertSecretName(vClusterName), metav1.GetOptions{})
					Expect(err).To(Succeed())

					By("Verifying CA cert is renewed after CA rotation", func() {
						caCertAfter, err := parseCertFromPEM(secret.Data[certs.CACertName])
						Expect(err).To(Succeed(), "parsing CA cert after CA rotation")
						Expect(certFingerprint(caCertAfter)).NotTo(Equal(caFingerprintBefore),
							"CA fingerprint should change after CA rotation")
						Expect(caCertAfter.NotAfter.After(caCertBefore.NotAfter)).To(BeTrue(),
							"CA cert should expire later after CA rotation")
					})

					By("Verifying apiserver cert is renewed after CA rotation", func() {
						apiserverCertAfter, err := parseCertFromPEM(secret.Data[certs.APIServerCertName])
						Expect(err).To(Succeed(), "parsing apiserver cert after CA rotation")
						Expect(certFingerprint(apiserverCertAfter)).NotTo(Equal(apiserverFingerprintBefore),
							"apiserver fingerprint should change after CA rotation")
						Expect(apiserverCertAfter.NotAfter.After(apiserverCertBefore.NotAfter)).To(BeTrue(),
							"apiserver cert should expire later after CA rotation")
					})
				})

				It("should reconnect after rotation tests", func(ctx context.Context) {
					reconnectVCluster(ctx, vClusterName, vClusterNamespace)
				})
			})

			// ------------------------------------------------------------------
			// Section 2: Cert expiration recovery
			// Lifecycle: confirm valid -> issue 1s CA -> wait expire -> rotate-ca -> verify recovered
			// ------------------------------------------------------------------
			Context("cert expiration recovery", func() {
				var caFingerprintBefore string

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

				// After issuing a 1s CA, the pod will be Running but NOT Ready because
				// the cert expires before the container can fully initialize. We only
				// check Phase == Running here (matching the old test behaviour), then
				// wait for expiry, then rotate-ca to recover.
				It("should have all pods running after short-lived CA rotation", func(ctx context.Context) {
					Eventually(func(g Gomega) {
						pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
							LabelSelector: "app=vcluster,release=" + vClusterName,
						})
						g.Expect(err).To(Succeed(), "listing vcluster pods")
						g.Expect(pods.Items).NotTo(BeEmpty(), "no vcluster pods found")
						for _, pod := range pods.Items {
							g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning),
								"pod %s phase: %s", pod.Name, pod.Status.Phase)
						}
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
				})

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

				It("should rotate the expired CA cert to recover", func(_ context.Context) {
					certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vClusterNamespace})
					certsCmd.SetArgs([]string{"rotate-ca", vClusterName})
					Expect(certsCmd.Execute()).To(Succeed())
				})

				It("should have all pods ready after rotating the expired CA", func(ctx context.Context) {
					waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutVeryLong)
				})

				It("should report new cert expiry via vcluster certs check after recovery", func(_ context.Context) {
					certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vClusterNamespace})
					certsCmd.SetArgs([]string{"check", vClusterName})
					Expect(certsCmd.Execute()).To(Succeed())
				})

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

				It("should reconnect after expiration recovery", func(ctx context.Context) {
					reconnectVCluster(ctx, vClusterName, vClusterNamespace)
				})
			})

			// ------------------------------------------------------------------
			// Section 3: Kubeconfig TLS behaviour after rotation
			// Lifecycle: capture baseline TLS -> rotate leaf -> verify old TLS works ->
			// rotate CA -> verify old TLS fails with CertificateVerificationError
			// ------------------------------------------------------------------
			Context("kubeconfig TLS behaviour", func() {
				var restConfigBefore *rest.Config

				It("should connect and capture baseline TLS config", func(ctx context.Context) {
					restConfig, vClusterClient := reconnectVCluster(ctx, vClusterName, vClusterNamespace)
					restConfigBefore = restConfig

					_, err := vClusterClient.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
					Expect(err).To(Succeed(), "baseline vCluster client should work")
				})

				It("should rotate the leaf certs", func(_ context.Context) {
					certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vClusterNamespace})
					certsCmd.SetArgs([]string{"rotate", vClusterName})
					Expect(certsCmd.Execute()).To(Succeed())
				})

				It("should have all pods ready after leaf rotation", func(ctx context.Context) {
					waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutVeryLong)
				})

				It("should still accept old TLS config after leaf rotation", func(ctx context.Context) {
					newRestConfig, _ := reconnectVCluster(ctx, vClusterName, vClusterNamespace)
					newRestConfig.TLSClientConfig = restConfigBefore.TLSClientConfig

					oldTLSClient, err := kubernetes.NewForConfig(newRestConfig)
					Expect(err).To(Succeed(), "building client with old TLS config after leaf rotation")

					_, err = oldTLSClient.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
					Expect(err).To(Succeed(),
						"old TLS config should remain valid after leaf-only rotation (CA unchanged)")
				})

				It("should rotate the CA cert", func(_ context.Context) {
					certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vClusterNamespace})
					certsCmd.SetArgs([]string{"rotate-ca", vClusterName})
					Expect(certsCmd.Execute()).To(Succeed())
				})

				It("should have all pods ready after CA rotation", func(ctx context.Context) {
					waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutVeryLong)
				})

				It("should reject old TLS config after CA rotation", func(ctx context.Context) {
					newRestConfig, _ := reconnectVCluster(ctx, vClusterName, vClusterNamespace)
					newRestConfig.TLSClientConfig = restConfigBefore.TLSClientConfig

					oldTLSClient, err := kubernetes.NewForConfig(newRestConfig)
					Expect(err).To(Succeed(), "building client with old TLS config")

					_, err = oldTLSClient.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})

					var certErr *tls.CertificateVerificationError
					Expect(errors.As(err, &certErr)).To(BeTrue(),
						"expected a TLS certificate verification error after CA rotation, got: %v", err)
				})

				It("should reconnect after all TLS tests", func(ctx context.Context) {
					reconnectVCluster(ctx, vClusterName, vClusterNamespace)
				})
			})
		},
	)
}
