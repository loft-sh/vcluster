package certs

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
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

		ginkgo.It("should successfully restart the vCluster pod", func() {
			gomega.Eventually(func() error {
				pods, err := f.HostClient.CoreV1().Pods(f.VClusterNamespace).List(f.Context, metav1.ListOptions{
					LabelSelector: "app=vcluster,release=" + f.VClusterName,
				})
				framework.ExpectNoError(err)

				for _, pod := range pods.Items {
					if len(pod.Status.ContainerStatuses) == 0 {
						return fmt.Errorf("pod %s has no container status", pod.Name)
					}

					for _, container := range pod.Status.ContainerStatuses {
						if container.State.Running == nil || !container.Ready {
							return fmt.Errorf("pod %s container %s is not running", pod.Name, container.Name)
						}
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

	ginkgo.It("should wait until the virtual cluster is ready again", func() {
		framework.ExpectNoError(f.WaitForVClusterReady())
	})

	ginkgo.Context("vCluster \"certs rotate-ca\"", ginkgo.Ordered, func() {
		ginkgo.It("should execute \"certs rotate-ca\" command", func() {
			certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: f.VClusterNamespace})
			certsCmd.SetArgs([]string{"rotate-ca", f.VClusterName})
			err = certsCmd.Execute()
			framework.ExpectNoError(err)
		})

		ginkgo.It("should successfully restart the vCluster pod", func() {
			gomega.Eventually(func() error {
				pods, err := f.HostClient.CoreV1().Pods(f.VClusterNamespace).List(f.Context, metav1.ListOptions{
					LabelSelector: "app=vcluster,release=" + f.VClusterName,
				})
				framework.ExpectNoError(err)

				for _, pod := range pods.Items {
					if len(pod.Status.ContainerStatuses) == 0 {
						return fmt.Errorf("pod %s has no container status", pod.Name)
					}

					for _, container := range pod.Status.ContainerStatuses {
						if container.State.Running == nil || !container.Ready {
							return fmt.Errorf("pod %s container %s is not running", pod.Name, container.Name)
						}
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

	ginkgo.It("should wait until the virtual cluster is ready again", func() {
		framework.ExpectNoError(f.WaitForVClusterReady())
	})

	ginkgo.AfterAll(func() {
		// Wait for virtual cluster to be ready after cert rotation and refresh the virtual client.
		framework.ExpectNoError(f.WaitForVClusterReady())
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
