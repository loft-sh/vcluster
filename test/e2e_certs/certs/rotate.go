package certs

import (
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
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var _ = ginkgo.Describe("vCluster cert rotation tests", ginkgo.Ordered, func() {
	var (
		f                          *framework.Framework
		secret                     *corev1.Secret
		err                        error
		apiserverCertBefore        *x509.Certificate
		apiserverFingerprintBefore string
		caCertBefore               *x509.Certificate
		caFingerprintBefore        string
	)

	ginkgo.JustBeforeEach(func() {
		f = framework.DefaultFramework
	})

	ginkgo.It("should obtain the current cert secret", func() {
		secret, err = f.HostClient.CoreV1().Secrets(f.VClusterNamespace).Get(f.Context, certs.CertSecretName(f.VClusterName), metav1.GetOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.It("should get the fingerprints from the cert secret", func() {
		apiserverCertBefore, err = parseCertFromPEM(secret.Data[certs.APIServerCertName])
		framework.ExpectNoError(err)
		apiserverFingerprintBefore = certFingerprint(apiserverCertBefore)

		caCertBefore, err = parseCertFromPEM(secret.Data[certs.CACertName])
		framework.ExpectNoError(err)
		caFingerprintBefore = certFingerprint(caCertBefore)
	})

	ginkgo.Context("vCluster \"certs rotate\"", ginkgo.Ordered, func() {
		ginkgo.It("should execute \"certs rotate\" command", func() {
			certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: f.VClusterNamespace})
			certsCmd.SetArgs([]string{"rotate", f.VClusterName})
			err = certsCmd.Execute()
			framework.ExpectNoError(err)
		})

		ginkgo.It("should wait until the virtual cluster is ready again", func() {
			framework.ExpectNoError(f.WaitForVClusterReady())
			gomega.Eventually(func(g gomega.Gomega) error {
				pods, err := f.HostClient.CoreV1().Pods(f.VClusterNamespace).List(f.Context, metav1.ListOptions{
					LabelSelector: "app=vcluster,release=" + f.VClusterName,
				})
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(pods.Items).NotTo(gomega.BeEmpty())

				for _, pod := range pods.Items {
					g.Expect(pod.Status.ContainerStatuses).NotTo(gomega.BeEmpty(),
						"pod %s should have container statuses", pod.Name)

					for i, container := range pod.Status.ContainerStatuses {
						g.Expect(container.State.Running).NotTo(gomega.BeNil(),
							"container %d in pod %s should be running", i, pod.Name)
						g.Expect(container.Ready).To(gomega.BeTrue(),
							"container %d in pod %s should be ready", i, pod.Name)
					}
				}

				return nil
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeoutLong).
				Should(gomega.Succeed())
		})

		ginkgo.It("should obtain the certs secret with new certificates", func() {
			gomega.Eventually(func() error {
				secret, err = f.HostClient.CoreV1().Secrets(f.VClusterNamespace).Get(f.Context, certs.CertSecretName(f.VClusterName), metav1.GetOptions{})
				if err != nil {
					return err
				}

				return nil
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeout).
				Should(gomega.Succeed())
		})

		ginkgo.It("should check that the CA certificate fingerprint and its expiry time did not change", func() {
			certAfter, err := parseCertFromPEM(secret.Data[certs.CACertName])
			framework.ExpectNoError(err)

			fingerprintAfter := certFingerprint(certAfter)

			// fingerprint should be equal.
			gomega.Expect(caFingerprintBefore).To(gomega.Equal(fingerprintAfter))

			// expiry date should be equal.
			gomega.Expect(certAfter.NotAfter).To(gomega.Equal(caCertBefore.NotAfter))

			// save fingerprint for next round
			caFingerprintBefore = fingerprintAfter
		})

		ginkgo.It("should check that the apiservcer certificate fingerprint is different and that it expires later", func() {
			apiserverCertAfter, err := parseCertFromPEM(secret.Data[certs.APIServerCertName])
			framework.ExpectNoError(err)

			fingerprintAfter := certFingerprint(apiserverCertAfter)

			// fingerprint should be different.
			gomega.Expect(apiserverFingerprintBefore).ToNot(gomega.Equal(fingerprintAfter))

			// new certificate should expire later than the old one.
			gomega.Expect(apiserverCertAfter.NotAfter.After(apiserverCertBefore.NotAfter)).To(gomega.BeTrue())

			// save fingerprint for next round
			apiserverFingerprintBefore = fingerprintAfter
		})
	})

	ginkgo.Context("vCluster \"certs rotate-ca\"", ginkgo.Ordered, func() {
		ginkgo.It("should execute \"certs rotate-ca\" command", func() {
			certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: f.VClusterNamespace})
			certsCmd.SetArgs([]string{"rotate-ca", f.VClusterName})
			err = certsCmd.Execute()
			framework.ExpectNoError(err)
		})

		ginkgo.It("should wait until the virtual cluster is ready again", func() {
			framework.ExpectNoError(f.WaitForVClusterReady())
			gomega.Eventually(func(g gomega.Gomega) error {
				pods, err := f.HostClient.CoreV1().Pods(f.VClusterNamespace).List(f.Context, metav1.ListOptions{
					LabelSelector: "app=vcluster,release=" + f.VClusterName,
				})
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(pods.Items).NotTo(gomega.BeEmpty())

				for _, pod := range pods.Items {
					g.Expect(pod.Status.ContainerStatuses).NotTo(gomega.BeEmpty(),
						"pod %s should have container statuses", pod.Name)

					for i, container := range pod.Status.ContainerStatuses {
						g.Expect(container.State.Running).NotTo(gomega.BeNil(),
							"container %d in pod %s should be running", i, pod.Name)
						g.Expect(container.Ready).To(gomega.BeTrue(),
							"container %d in pod %s should be ready", i, pod.Name)
					}
				}

				return nil
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeoutLong).
				Should(gomega.Succeed())
		})

		ginkgo.It("should obtain the current cert secret", func() {
			secret, err = f.HostClient.CoreV1().Secrets(f.VClusterNamespace).Get(f.Context, certs.CertSecretName(f.VClusterName), metav1.GetOptions{})
			framework.ExpectNoError(err)
		})

		ginkgo.It("should check that the CA and apiserver certificate fingerprints are different and that they expire later", func() {
			apiserverCertAfter, err := parseCertFromPEM(secret.Data[certs.APIServerCertName])
			framework.ExpectNoError(err)
			caCertAfter, err := parseCertFromPEM(secret.Data[certs.CACertName])
			framework.ExpectNoError(err)

			apiserverFingerprintAfter := certFingerprint(apiserverCertAfter)
			caFingerprintAfter := certFingerprint(caCertAfter)

			// fingerprints should be different.
			gomega.Expect(apiserverFingerprintBefore).ToNot(gomega.Equal(apiserverFingerprintAfter))
			gomega.Expect(caFingerprintBefore).ToNot(gomega.Equal(caFingerprintAfter))

			// new certificates should expire later than the old ones.
			gomega.Expect(apiserverCertAfter.NotAfter.After(apiserverCertBefore.NotAfter)).To(gomega.BeTrue())
			gomega.Expect(caCertAfter.NotAfter.After(caCertBefore.NotAfter)).To(gomega.BeTrue())
		})
	})

	ginkgo.AfterAll(func() {
		framework.ExpectNoError(f.RefreshVirtualClient())
	})
})

var _ = ginkgo.Describe("vCluster cert rotation expiration tests", ginkgo.Ordered, func() {
	var (
		f *framework.Framework
	)

	ginkgo.JustBeforeEach(func() {
		f = framework.DefaultFramework
	})

	ginkgo.It("should obtain the current cert secret of vCluster", func() {
		_, err := f.HostClient.CoreV1().Secrets(f.VClusterNamespace).Get(f.Context, certs.CertSecretName(f.VClusterName), metav1.GetOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.It("checking current validity date of CA cert of vCluster", func() {
		secret, err := f.HostClient.CoreV1().Secrets(f.VClusterNamespace).Get(
			f.Context, certs.CertSecretName(f.VClusterName), metav1.GetOptions{})
		framework.ExpectNoError(err)

		certPEM := secret.Data["ca.crt"]

		block, _ := pem.Decode(certPEM)
		gomega.Expect(block).NotTo(gomega.BeNil(), "Failed to decode PEM block")

		cert, err := x509.ParseCertificate(block.Bytes)
		framework.ExpectNoError(err)

		gomega.Expect(cert.NotAfter.After(time.Now())).To(gomega.BeTrue(), "CA cert is valid")
	})

	ginkgo.It("setting validity of ca cert of vCluster to 1 seconds", func() {
		os.Setenv("DEVELOPMENT", "true")
		os.Setenv("VCLUSTER_CERTS_VALIDITYPERIOD", "1s")
		defer os.Unsetenv("DEVELOPMENT")
		defer os.Unsetenv("VCLUSTER_CERTS_VALIDITYPERIOD")

		certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: f.VClusterNamespace})
		certsCmd.SetArgs([]string{"rotate-ca", f.VClusterName})

		err := certsCmd.Execute()
		framework.ExpectNoError(err)
	})

	ginkgo.It("should wait until the vCluster is ready again", func() {
		framework.ExpectNoError(f.WaitForVClusterReady())
		gomega.Eventually(func(g gomega.Gomega) error {
			pods, err := f.HostClient.CoreV1().Pods(f.VClusterNamespace).List(f.Context, metav1.ListOptions{
				LabelSelector: "app=vcluster,release=" + f.VClusterName,
			})
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(pods.Items).NotTo(gomega.BeEmpty())

			for _, pod := range pods.Items {
				g.Expect(pod.Status.ContainerStatuses).NotTo(gomega.BeEmpty(),
					"pod %s should have container statuses", pod.Name)

				for i, container := range pod.Status.ContainerStatuses {
					g.Expect(container.State.Running).NotTo(gomega.BeNil(),
						"container %d in pod %s should be running", i, pod.Name)
					g.Expect(container.Ready).To(gomega.BeTrue(),
						"container %d in pod %s should be ready", i, pod.Name)
				}
			}
			return nil
		}).WithPolling(time.Second).
			WithTimeout(framework.PollTimeoutLong).
			Should(gomega.Succeed())
	})

	ginkgo.It("should check if CA cert of vCluster is expired", func() {
		gomega.Eventually(func(g gomega.Gomega) error {
			secret, err := f.HostClient.CoreV1().Secrets(f.VClusterNamespace).Get(
				f.Context, certs.CertSecretName(f.VClusterName), metav1.GetOptions{})
			g.Expect(err).NotTo(gomega.HaveOccurred())

			certPEM := secret.Data["ca.crt"]
			block, _ := pem.Decode(certPEM)
			g.Expect(block).NotTo(gomega.BeNil())

			cert, err := x509.ParseCertificate(block.Bytes)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			if cert.NotAfter.Before(time.Now()) {
				return nil
			}
			return fmt.Errorf("CA cert not expired yet (expires at %s)", cert.NotAfter)
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeoutLong).
			Should(gomega.Succeed())
	})

	ginkgo.It("priniting expired status of vCluster CA cert", func() {
		certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: f.VClusterNamespace})
		certsCmd.SetArgs([]string{"check", f.VClusterName})

		err := certsCmd.Execute()
		framework.ExpectNoError(err)
	})

	ginkgo.It("rotating expired CA cert of vCluster", func() {
		certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: f.VClusterNamespace})
		certsCmd.SetArgs([]string{"rotate-ca", f.VClusterName})

		err := certsCmd.Execute()
		framework.ExpectNoError(err)
	})

	ginkgo.It("should wait until the vCluster is ready again", func() {
		framework.ExpectNoError(f.WaitForVClusterReady())
		gomega.Eventually(func(g gomega.Gomega) error {
			pods, err := f.HostClient.CoreV1().Pods(f.VClusterNamespace).List(f.Context, metav1.ListOptions{
				LabelSelector: "app=vcluster,release=" + f.VClusterName,
			})
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(pods.Items).NotTo(gomega.BeEmpty())

			for _, pod := range pods.Items {
				g.Expect(pod.Status.ContainerStatuses).NotTo(gomega.BeEmpty(),
					"pod %s should have container statuses", pod.Name)

				for i, container := range pod.Status.ContainerStatuses {
					g.Expect(container.State.Running).NotTo(gomega.BeNil(),
						"container %d in pod %s should be running", i, pod.Name)
					g.Expect(container.Ready).To(gomega.BeTrue(),
						"container %d in pod %s should be ready", i, pod.Name)
				}
			}
			return nil
		}).WithPolling(time.Second).
			WithTimeout(framework.PollTimeoutLong).
			Should(gomega.Succeed())
	})

	ginkgo.It("priniting new expiry date and time of vCluster CA cert", func() {
		certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: f.VClusterNamespace})
		certsCmd.SetArgs([]string{"check", f.VClusterName})

		err := certsCmd.Execute()
		framework.ExpectNoError(err)
	})

	ginkgo.It("checking new validity date of CA cert of vCluster", func() {
		secret, err := f.HostClient.CoreV1().Secrets(f.VClusterNamespace).Get(
			f.Context, certs.CertSecretName(f.VClusterName), metav1.GetOptions{})
		framework.ExpectNoError(err)

		certPEM := secret.Data["ca.crt"]
		block, _ := pem.Decode(certPEM)
		gomega.Expect(block).NotTo(gomega.BeNil(), "Failed to decode PEM block")

		cert, err := x509.ParseCertificate(block.Bytes)
		framework.ExpectNoError(err)

		gomega.Expect(cert.NotAfter.After(time.Now())).To(gomega.BeTrue(), "CA cert is valid")
	})

	ginkgo.AfterAll(func() {
		framework.ExpectNoError(f.RefreshVirtualClient())
	})
})

var _ = ginkgo.Describe("vCluster cert rotation kube config tests", ginkgo.Ordered, func() {
	var (
		f                *framework.Framework
		restConfigBefore *rest.Config
	)

	ginkgo.JustBeforeEach(func() {
		f = framework.DefaultFramework
	})

	ginkgo.It("should be able to use the virtual client", func() {
		_, err := f.VClusterClient.CoreV1().Pods(corev1.NamespaceDefault).List(f.Context, metav1.ListOptions{})
		framework.ExpectNoError(err)

		restConfigBefore = f.VClusterConfig
		vClusterClient, err := kubernetes.NewForConfig(restConfigBefore)
		framework.ExpectNoError(err)

		_, err = vClusterClient.CoreV1().Pods(corev1.NamespaceDefault).List(f.Context, metav1.ListOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.Context("vCluster \"certs rotate\"", ginkgo.Ordered, func() {
		ginkgo.It("should execute \"certs rotate\" command", func() {
			certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: f.VClusterNamespace})
			certsCmd.SetArgs([]string{"rotate", f.VClusterName})
			framework.ExpectNoError(certsCmd.Execute())
		})

		ginkgo.It("should wait until the virtual cluster is ready again", func() {
			framework.ExpectNoError(f.WaitForVClusterReady())
			gomega.Eventually(func(g gomega.Gomega) error {
				pods, err := f.HostClient.CoreV1().Pods(f.VClusterNamespace).List(f.Context, metav1.ListOptions{
					LabelSelector: "app=vcluster,release=" + f.VClusterName,
				})
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(pods.Items).NotTo(gomega.BeEmpty())

				for _, pod := range pods.Items {
					g.Expect(pod.Status.ContainerStatuses).NotTo(gomega.BeEmpty(),
						"pod %s should have container statuses", pod.Name)

					for i, container := range pod.Status.ContainerStatuses {
						g.Expect(container.State.Running).NotTo(gomega.BeNil(),
							"container %d in pod %s should be running", i, pod.Name)
						g.Expect(container.Ready).To(gomega.BeTrue(),
							"container %d in pod %s should be ready", i, pod.Name)
					}
				}

				return nil
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeoutLong).
				Should(gomega.Succeed())
		})

		ginkgo.It("should not receive a tls verification error using the old tls config for the virtual client", func() {
			framework.ExpectNoError(f.RefreshVirtualClient())

			cfg := f.VClusterConfig
			cfg.TLSClientConfig = restConfigBefore.TLSClientConfig

			vClusterClient, err := kubernetes.NewForConfig(cfg)
			framework.ExpectNoError(err)

			_, err = vClusterClient.CoreV1().Pods(corev1.NamespaceDefault).List(f.Context, metav1.ListOptions{})
			framework.ExpectNoError(err)
		})
	})

	ginkgo.Context("vCluster \"certs rotate-ca\"", ginkgo.Ordered, func() {
		ginkgo.It("should execute \"certs rotate-ca\" command", func() {
			certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: f.VClusterNamespace})
			certsCmd.SetArgs([]string{"rotate-ca", f.VClusterName})
			framework.ExpectNoError(certsCmd.Execute())
		})

		ginkgo.It("should wait until the virtual cluster is ready again", func() {
			framework.ExpectNoError(f.WaitForVClusterReady())
			gomega.Eventually(func(g gomega.Gomega) error {
				pods, err := f.HostClient.CoreV1().Pods(f.VClusterNamespace).List(f.Context, metav1.ListOptions{
					LabelSelector: "app=vcluster,release=" + f.VClusterName,
				})
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(pods.Items).NotTo(gomega.BeEmpty())

				for _, pod := range pods.Items {
					g.Expect(pod.Status.ContainerStatuses).NotTo(gomega.BeEmpty(),
						"pod %s should have container statuses", pod.Name)

					for i, container := range pod.Status.ContainerStatuses {
						g.Expect(container.State.Running).NotTo(gomega.BeNil(),
							"container %d in pod %s should be running", i, pod.Name)
						g.Expect(container.Ready).To(gomega.BeTrue(),
							"container %d in pod %s should be ready", i, pod.Name)
					}
				}

				return nil
			}).WithPolling(time.Second).
				WithTimeout(framework.PollTimeoutLong).
				Should(gomega.Succeed())
		})

		ginkgo.It("should receive a tls verification error using the old tls config for the virtual client", func() {
			framework.ExpectNoError(f.RefreshVirtualClient())

			cfg := f.VClusterConfig
			cfg.TLSClientConfig = restConfigBefore.TLSClientConfig

			vClusterClient, err := kubernetes.NewForConfig(cfg)
			framework.ExpectNoError(err)

			_, err = vClusterClient.CoreV1().Pods(corev1.NamespaceDefault).List(f.Context, metav1.ListOptions{})
			framework.ExpectError(err)

			var certErr *tls.CertificateVerificationError
			if !errors.As(err, &certErr) {
				framework.Failf("received non-tls verification error: %v", err)
			}
		})
	})

	ginkgo.AfterAll(func() {
		framework.ExpectNoError(f.RefreshVirtualClient())
	})
})

func parseCertFromPEM(pemData []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("decoding to PEM block")
	}

	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("not a certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing certificate: %w", err)
	}

	return cert, nil
}

func certFingerprint(cert *x509.Certificate) string {
	hash := sha256.Sum256(cert.Raw)
	fingerprint := hex.EncodeToString(hash[:])

	// Format as colon-separated hex pairs (like OpenSSL output)
	var formatted strings.Builder
	for i := 0; i < len(fingerprint); i += 2 {
		if i > 0 {
			formatted.WriteString(":")
		}
		formatted.WriteString(strings.ToUpper(fingerprint[i : i+2]))
	}

	return formatted.String()
}
