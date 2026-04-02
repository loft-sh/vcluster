package certs

import (
	"context"
	"crypto/x509"

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

// DescribeCertRotation registers cert rotation tests (rotate + rotate-ca) with fingerprint
// verification against the given vCluster.
// Ordered because the specs form a lifecycle: get secret -> capture fingerprints ->
// rotate leaf -> verify fingerprints unchanged for CA, changed for leaf ->
// rotate CA -> verify both fingerprints changed.
func DescribeCertRotation(vcluster suite.Dependency) bool {
	return Describe("vCluster cert rotation",
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

				// Fingerprints captured before rotation to verify changes.
				apiserverCertBefore        *x509.Certificate
				apiserverFingerprintBefore string
				caCertBefore               *x509.Certificate
				caFingerprintBefore        string
			)

			BeforeAll(func(ctx context.Context) context.Context {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterNamespace = "vcluster-" + vClusterName
				return ctx
			})

			// Verify vCluster is ready and all pods are running.
			It("should have all vCluster pods running and ready", func(ctx context.Context) {
				waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutLong)
			})

			// Verify the cert secret exists before any rotation.
			It("should have the cert secret", func(ctx context.Context) {
				_, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
					certs.CertSecretName(vClusterName), metav1.GetOptions{})
				Expect(err).To(Succeed())
			})

			// Report cert expiry before any rotation.
			It("should report cert expiry via vcluster certs check", func(_ context.Context) {
				certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vClusterNamespace})
				certsCmd.SetArgs([]string{"check", vClusterName})
				Expect(certsCmd.Execute()).To(Succeed())
			})

			// Capture fingerprints before rotation to enable comparison later.
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

			// Rotate leaf certs (does not change CA).
			It("should rotate the leaf certs", func(_ context.Context) {
				certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vClusterNamespace})
				certsCmd.SetArgs([]string{"rotate", vClusterName})
				Expect(certsCmd.Execute()).To(Succeed())
			})

			// Wait for recovery after leaf rotation.
			It("should have all pods ready after leaf cert rotation", func(ctx context.Context) {
				waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutVeryLong)
			})

			// After leaf rotation: CA fingerprint and expiry must be unchanged;
			// apiserver fingerprint must differ and expire later.
			It("should verify CA cert is unchanged and leaf cert is renewed after leaf rotation", func(ctx context.Context) {
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

					// Update fingerprint for the next rotation comparison.
					caFingerprintBefore = certFingerprint(caCertAfter)
				})

				By("Verifying apiserver cert fingerprint is updated and expires later after leaf rotation", func() {
					apiserverCertAfter, err := parseCertFromPEM(secret.Data[certs.APIServerCertName])
					Expect(err).To(Succeed(), "parsing apiserver cert after leaf rotation")

					Expect(certFingerprint(apiserverCertAfter)).NotTo(Equal(apiserverFingerprintBefore),
						"apiserver fingerprint should change after leaf rotation")
					Expect(apiserverCertAfter.NotAfter.After(apiserverCertBefore.NotAfter)).To(BeTrue(),
						"apiserver cert should expire later after leaf rotation")

					// Update for next comparison.
					apiserverFingerprintBefore = certFingerprint(apiserverCertAfter)
				})
			})

			// Rotate the CA (rotates both CA and all leaf certs).
			It("should rotate the CA cert", func(_ context.Context) {
				certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vClusterNamespace})
				certsCmd.SetArgs([]string{"rotate-ca", vClusterName})
				Expect(certsCmd.Execute()).To(Succeed())
			})

			// Wait for recovery after CA rotation.
			It("should have all pods ready after CA rotation", func(ctx context.Context) {
				waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutVeryLong)
			})

			// After CA rotation: both CA and apiserver fingerprints must differ and expire later.
			It("should verify both CA and leaf certs are renewed after CA rotation", func(ctx context.Context) {
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

			// Reconnect after all rotations so subsequent test suites (or re-runs) get a live proxy.
			It("should reconnect to vCluster after all rotations", func(ctx context.Context) {
				reconnectVCluster(ctx, vClusterName, vClusterNamespace)
			})
		},
	)
}
