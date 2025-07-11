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

var _ = ginkgo.Describe("vCluster cert rotation tests", func() {
	f := framework.DefaultFramework

	ginkgo.Context("should execute certs rotate command", func() {
		var (
			secret            *corev1.Secret
			err               error
			certBefore        *x509.Certificate
			fingerprintBefore string
		)

		ginkgo.It("should obtain the current cert secret", func() {
			secret, err = f.HostClient.CoreV1().Secrets(f.VClusterNamespace).Get(f.Context, certs.CertSecretName(f.VClusterName), metav1.GetOptions{})
			framework.ExpectNoError(err)
		})

		ginkgo.It("should get a cert fingerprint from the cert secret", func() {
			certBefore, err = parseCertFromPEM(secret.Data[certs.APIServerCertName])
			framework.ExpectNoError(err)
			fingerprintBefore = certFingerprint(certBefore)
		})

		ginkgo.It("should execute certs rotate command", func() {
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
				WithTimeout(framework.PollTimeout).
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

		ginkgo.It("should check that the certificate fingerprint and expiry dates are different", func() {
			certAfter, err := parseCertFromPEM(secret.Data[certs.APIServerCertName])
			framework.ExpectNoError(err)

			fingerprintAfter := certFingerprint(certAfter)

			// fingerprint should be different.
			gomega.Expect(fingerprintBefore).ToNot(gomega.Equal(fingerprintAfter))

			// new certificate should expire later than the old one.
			gomega.Expect(certAfter.NotAfter.After(certBefore.NotAfter)).To(gomega.BeTrue())
		})
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
