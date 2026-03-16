package test_certs

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	certscmd "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/certs"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/flags"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	loftlog "github.com/loft-sh/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Ordered justification: cert rotation is destructive and cumulative —
// each context modifies the vCluster's certificates, and subsequent
// contexts depend on the state left by prior ones.
var _ = Describe("Certificate Rotation",
	Ordered,
	labels.PR,
	labels.Certs,
	cluster.Use(clusters.CertsVCluster),
	cluster.Use(clusters.HostCluster),
	func() {
		var (
			hostClient    kubernetes.Interface
			vclusterNS    = "vcluster-" + clusters.CertsVClusterName
			vclusterName  = clusters.CertsVClusterName
			labelSelector = "app=vcluster,release=" + clusters.CertsVClusterName

			// Fingerprints tracked across contexts
			apiserverFingerprintBefore string
			apiserverCertBefore        *x509.Certificate
			caFingerprintBefore        string
			caCertBefore               *x509.Certificate
		)

		BeforeAll(func(ctx context.Context) {
			By("Obtaining host client and initial certificate fingerprints", func() {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())

				secret, err := hostClient.CoreV1().Secrets(vclusterNS).Get(ctx, certs.CertSecretName(vclusterName), metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred(), "should obtain the initial cert secret")

				apiserverCertBefore, err = parseCertFromPEM(secret.Data[certs.APIServerCertName])
				Expect(err).NotTo(HaveOccurred(), "should parse apiserver cert")
				apiserverFingerprintBefore = certFingerprint(apiserverCertBefore)

				caCertBefore, err = parseCertFromPEM(secret.Data[certs.CACertName])
				Expect(err).NotTo(HaveOccurred(), "should parse CA cert")
				caFingerprintBefore = certFingerprint(caCertBefore)
			})
		})

		Context("certs rotate", func() {
			It("should preserve CA cert and rotate apiserver cert", func(ctx context.Context) {
				By("Executing certs rotate command", func() {
					certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vclusterNS})
					certsCmd.SetArgs([]string{"rotate", vclusterName})
					Expect(certsCmd.Execute()).To(Succeed())
				})

				By("Waiting for vCluster pods to be ready again", func() {
					waitForVClusterReady(ctx, hostClient, vclusterNS, labelSelector)
				})

				By("Verifying CA fingerprint and expiry are unchanged and apiserver fingerprint changed", func() {
					secret, err := hostClient.CoreV1().Secrets(vclusterNS).Get(ctx, certs.CertSecretName(vclusterName), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					caCertAfter, err := parseCertFromPEM(secret.Data[certs.CACertName])
					Expect(err).NotTo(HaveOccurred())
					caFingerprintAfter := certFingerprint(caCertAfter)
					Expect(caFingerprintAfter).To(Equal(caFingerprintBefore), "CA fingerprint should be unchanged after leaf rotation")
					Expect(caCertAfter.NotAfter).To(Equal(caCertBefore.NotAfter), "CA expiry should be unchanged after leaf rotation")

					apiserverCertAfter, err := parseCertFromPEM(secret.Data[certs.APIServerCertName])
					Expect(err).NotTo(HaveOccurred())
					apiserverFingerprintAfter := certFingerprint(apiserverCertAfter)
					Expect(apiserverFingerprintAfter).NotTo(Equal(apiserverFingerprintBefore), "apiserver fingerprint should change after rotation")
					Expect(apiserverCertAfter.NotAfter.After(apiserverCertBefore.NotAfter)).To(BeTrue(), "new apiserver cert should expire later")

					// Update tracked fingerprints for subsequent contexts
					caFingerprintBefore = caFingerprintAfter
					apiserverFingerprintBefore = apiserverFingerprintAfter
					apiserverCertBefore = apiserverCertAfter
				})
			})
		})

		Context("certs rotate-ca", func() {
			It("should rotate both CA and apiserver certs", func(ctx context.Context) {
				By("Executing certs rotate-ca command", func() {
					certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vclusterNS})
					certsCmd.SetArgs([]string{"rotate-ca", vclusterName})
					Expect(certsCmd.Execute()).To(Succeed())
				})

				By("Waiting for vCluster pods to be ready again", func() {
					waitForVClusterReady(ctx, hostClient, vclusterNS, labelSelector)
				})

				By("Verifying both CA and apiserver fingerprints changed and expire later", func() {
					secret, err := hostClient.CoreV1().Secrets(vclusterNS).Get(ctx, certs.CertSecretName(vclusterName), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					caCertAfter, err := parseCertFromPEM(secret.Data[certs.CACertName])
					Expect(err).NotTo(HaveOccurred())
					caFingerprintAfter := certFingerprint(caCertAfter)
					Expect(caFingerprintAfter).NotTo(Equal(caFingerprintBefore), "CA fingerprint should change after CA rotation")
					Expect(caCertAfter.NotAfter.After(caCertBefore.NotAfter)).To(BeTrue(), "new CA cert should expire later")

					apiserverCertAfter, err := parseCertFromPEM(secret.Data[certs.APIServerCertName])
					Expect(err).NotTo(HaveOccurred())
					apiserverFingerprintAfter := certFingerprint(apiserverCertAfter)
					Expect(apiserverFingerprintAfter).NotTo(Equal(apiserverFingerprintBefore), "apiserver fingerprint should change after CA rotation")
					Expect(apiserverCertAfter.NotAfter.After(apiserverCertBefore.NotAfter)).To(BeTrue(), "new apiserver cert should expire later")

					// Update tracked state
					caCertBefore = caCertAfter
					caFingerprintBefore = caFingerprintAfter
					apiserverCertBefore = apiserverCertAfter
					apiserverFingerprintBefore = apiserverFingerprintAfter
				})
			})
		})

		Context("expired certificate rotation", func() {
			It("should rotate expired certificates", func(ctx context.Context) {
				By("Verifying current CA cert is valid", func() {
					secret, err := hostClient.CoreV1().Secrets(vclusterNS).Get(ctx, certs.CertSecretName(vclusterName), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					caCert, err := parseCertFromPEM(secret.Data[certs.CACertName])
					Expect(err).NotTo(HaveOccurred())
					Expect(caCert.NotAfter.After(time.Now())).To(BeTrue(), "CA cert should currently be valid")
				})

				By("Setting 1-second validity and rotating CA to create soon-to-expire certs", func() {
					Expect(os.Setenv("DEVELOPMENT", "true")).To(Succeed())
					Expect(os.Setenv("VCLUSTER_CERTS_VALIDITYPERIOD", "1s")).To(Succeed())
					defer func() {
						os.Unsetenv("DEVELOPMENT")
						os.Unsetenv("VCLUSTER_CERTS_VALIDITYPERIOD")
					}()

					certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vclusterNS})
					certsCmd.SetArgs([]string{"rotate-ca", vclusterName})
					Expect(certsCmd.Execute()).To(Succeed())
				})

				By("Waiting for vCluster pods to be running", func() {
					Eventually(func(g Gomega) {
						pods, err := hostClient.CoreV1().Pods(vclusterNS).List(ctx, metav1.ListOptions{
							LabelSelector: labelSelector,
						})
						g.Expect(err).NotTo(HaveOccurred(), "should list vCluster pods")
						g.Expect(pods.Items).NotTo(BeEmpty(), "should have at least one vCluster pod")
						for _, pod := range pods.Items {
							g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning),
								"pod %s should be running, got %s", pod.Name, pod.Status.Phase)
						}
					}).WithPolling(constants.PollingInterval).
						WithTimeout(constants.PollingTimeoutLong).
						Should(Succeed())
				})

				By("Waiting for CA cert to expire", func() {
					Eventually(func(g Gomega) {
						secret, err := hostClient.CoreV1().Secrets(vclusterNS).Get(ctx, certs.CertSecretName(vclusterName), metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred())

						block, _ := pem.Decode(secret.Data[certs.CACertName])
						g.Expect(block).NotTo(BeNil(), "should decode PEM block")

						cert, err := x509.ParseCertificate(block.Bytes)
						g.Expect(err).NotTo(HaveOccurred())
						g.Expect(cert.NotAfter.Before(time.Now())).To(BeTrue(),
							"CA cert should be expired, but expires at %s", cert.NotAfter)
					}).WithPolling(constants.PollingInterval).
						WithTimeout(constants.PollingTimeoutLong).
						Should(Succeed())
				})

				By("Rotating expired CA with normal validity", func() {
					certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vclusterNS})
					certsCmd.SetArgs([]string{"rotate-ca", vclusterName})
					Expect(certsCmd.Execute()).To(Succeed())
				})

				By("Waiting for vCluster to be fully ready", func() {
					waitForVClusterReady(ctx, hostClient, vclusterNS, labelSelector)
				})

				By("Verifying new CA cert is valid", func() {
					secret, err := hostClient.CoreV1().Secrets(vclusterNS).Get(ctx, certs.CertSecretName(vclusterName), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					caCert, err := parseCertFromPEM(secret.Data[certs.CACertName])
					Expect(err).NotTo(HaveOccurred())
					Expect(caCert.NotAfter.After(time.Now())).To(BeTrue(), "new CA cert should be valid")

					// Update tracked state
					caCertBefore = caCert
					caFingerprintBefore = certFingerprint(caCert)
				})
			})
		})

		Context("kube config compatibility", func() {
			var restConfigBefore *rest.Config

			BeforeAll(func(ctx context.Context) {
				By("Reconnecting to vCluster and saving current TLS config", func() {
					cfg, cleanup := connectVCluster(ctx, vclusterName, vclusterNS)
					DeferCleanup(cleanup)

					vClient, err := kubernetes.NewForConfig(cfg)
					Expect(err).NotTo(HaveOccurred())
					_, err = vClient.CoreV1().Pods(corev1.NamespaceDefault).List(ctx, metav1.ListOptions{})
					Expect(err).NotTo(HaveOccurred(), "should be able to list pods with current config")

					restConfigBefore = rest.CopyConfig(cfg)
				})
			})

			Context("after certs rotate", func() {
				It("should allow old TLS config after leaf rotation", func(ctx context.Context) {
					By("Executing certs rotate command", func() {
						certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vclusterNS})
						certsCmd.SetArgs([]string{"rotate", vclusterName})
						Expect(certsCmd.Execute()).To(Succeed())
					})

					By("Waiting for vCluster pods to be ready", func() {
						waitForVClusterReady(ctx, hostClient, vclusterNS, labelSelector)
					})

					By("Verifying old TLS config still works after leaf rotation", func() {
						cfg, cleanup := connectVCluster(ctx, vclusterName, vclusterNS)
						DeferCleanup(cleanup)

						// Use new connection's server address but old TLS config
						cfg.TLSClientConfig = restConfigBefore.TLSClientConfig
						vClient, err := kubernetes.NewForConfig(cfg)
						Expect(err).NotTo(HaveOccurred())

						_, err = vClient.CoreV1().Pods(corev1.NamespaceDefault).List(ctx, metav1.ListOptions{})
						Expect(err).NotTo(HaveOccurred(), "old TLS config should still work after leaf rotation (CA unchanged)")
					})
				})
			})

			Context("after certs rotate-ca", func() {
				It("should reject old TLS config after CA rotation", func(ctx context.Context) {
					By("Executing certs rotate-ca command", func() {
						certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: vclusterNS})
						certsCmd.SetArgs([]string{"rotate-ca", vclusterName})
						Expect(certsCmd.Execute()).To(Succeed())
					})

					By("Waiting for vCluster pods to be ready", func() {
						waitForVClusterReady(ctx, hostClient, vclusterNS, labelSelector)
					})

					By("Verifying old TLS config fails after CA rotation", func() {
						cfg, cleanup := connectVCluster(ctx, vclusterName, vclusterNS)
						DeferCleanup(cleanup)

						// Use new connection's server address but old TLS config
						cfg.TLSClientConfig = restConfigBefore.TLSClientConfig
						vClient, err := kubernetes.NewForConfig(cfg)
						Expect(err).NotTo(HaveOccurred())

						_, err = vClient.CoreV1().Pods(corev1.NamespaceDefault).List(ctx, metav1.ListOptions{})
						Expect(err).To(HaveOccurred(), "old TLS config should fail after CA rotation")

						var certErr *tls.CertificateVerificationError
						Expect(errors.As(err, &certErr)).To(BeTrue(),
							"expected tls.CertificateVerificationError but got: %v", err)
					})
				})
			})
		})

		AfterAll(func(ctx context.Context) {
			By("Reconnecting to vCluster to restore suite proxy for subsequent tests", func() {
				_, cleanup := connectVCluster(ctx, vclusterName, vclusterNS)
				DeferCleanup(cleanup)
			})
		})
	},
)

// waitForVClusterReady polls until all vCluster pods are running with all containers ready.
func waitForVClusterReady(ctx context.Context, hostClient kubernetes.Interface, namespace, labelSelector string) {
	Eventually(func(g Gomega) {
		pods, err := hostClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		g.Expect(err).NotTo(HaveOccurred(), "should list vCluster pods")
		g.Expect(pods.Items).NotTo(BeEmpty(), "should have at least one vCluster pod")

		for _, pod := range pods.Items {
			g.Expect(pod.Status.ContainerStatuses).NotTo(BeEmpty(),
				"pod %s should have container statuses", pod.Name)
			for i, container := range pod.Status.ContainerStatuses {
				g.Expect(container.State.Running).NotTo(BeNil(),
					"container %d in pod %s should be running", i, pod.Name)
				g.Expect(container.Ready).To(BeTrue(),
					"container %d in pod %s should be ready", i, pod.Name)
			}
		}
	}).WithPolling(constants.PollingInterval).
		WithTimeout(constants.PollingTimeoutLong).
		Should(Succeed())
}

// connectVCluster establishes a fresh connection to the vCluster using a background proxy.
// Returns a rest.Config and a cleanup function that removes the temp kubeconfig.
func connectVCluster(ctx context.Context, name, namespace string) (*rest.Config, func()) {
	tmpFile, err := os.CreateTemp("", "vcluster-kubeconfig-*.yaml")
	Expect(err).NotTo(HaveOccurred())
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	cleanupFn := func() {
		os.Remove(tmpPath)
	}

	options := &cli.ConnectOptions{
		BackgroundProxy:      true,
		BackgroundProxyImage: constants.GetVClusterImage(),
		KubeConfig:           tmpPath,
	}
	globalFlags := &flags.GlobalFlags{
		Namespace: namespace,
	}

	err = cli.ConnectHelm(ctx, options, globalFlags, name, nil, loftlog.Discard)
	Expect(err).NotTo(HaveOccurred(), "vcluster connect should succeed")

	var cfg *rest.Config
	Eventually(func(g Gomega) {
		data, err := os.ReadFile(tmpPath)
		g.Expect(err).NotTo(HaveOccurred(), "should read kubeconfig file")
		g.Expect(data).NotTo(BeEmpty(), "kubeconfig file should not be empty")

		cfg, err = clientcmd.RESTConfigFromKubeConfig(data)
		g.Expect(err).NotTo(HaveOccurred(), "should parse kubeconfig")
	}).WithPolling(constants.PollingInterval).
		WithTimeout(constants.PollingTimeout).
		Should(Succeed())

	// Verify connection works
	vClient, err := kubernetes.NewForConfig(cfg)
	Expect(err).NotTo(HaveOccurred())
	Eventually(func(g Gomega) {
		_, err := vClient.CoreV1().ServiceAccounts(corev1.NamespaceDefault).Get(ctx, "default", metav1.GetOptions{})
		g.Expect(err).NotTo(HaveOccurred(), "should be able to reach vCluster API")
	}).WithPolling(constants.PollingInterval).
		WithTimeout(constants.PollingTimeout).
		Should(Succeed())

	return cfg, cleanupFn
}

func parseCertFromPEM(pemData []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("decoding to PEM block")
	}
	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("not a certificate")
	}
	return x509.ParseCertificate(block.Bytes)
}

func certFingerprint(cert *x509.Certificate) string {
	hash := sha256.Sum256(cert.Raw)
	fingerprint := hex.EncodeToString(hash[:])
	var formatted strings.Builder
	for i := 0; i < len(fingerprint); i += 2 {
		if i > 0 {
			formatted.WriteString(":")
		}
		formatted.WriteString(strings.ToUpper(fingerprint[i : i+2]))
	}
	return formatted.String()
}
