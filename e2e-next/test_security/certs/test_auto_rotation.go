package certs

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/certs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CertAutoRotationSpec registers tests that verify automatic certificate
// rotation when leaf certs are near expiry. These tests are Ordered because
// they form a lifecycle: record state -> inject expiring cert -> restart -> verify.
//
// Must be called inside a Describe that has cluster.Use() for the vcluster and host cluster.
func CertAutoRotationSpec() {
	Describe("vCluster cert auto-rotation",
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

			It("should have all vCluster pods running and ready", func(ctx context.Context) {
				waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutLong)
			})

			var originalCANotAfter time.Time

			// Spec 2 depends on 1: record original CA and inject an expiring leaf cert
			It("should inject an expiring leaf cert into the secret", func(ctx context.Context) {
				By("Recording the original CA cert NotAfter", func() {
					secret, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
						certs.CertSecretName(vClusterName), metav1.GetOptions{})
					Expect(err).To(Succeed())

					block, _ := pem.Decode(secret.Data["ca.crt"])
					Expect(block).NotTo(BeNil(), "failed to decode CA cert PEM")
					ca, err := x509.ParseCertificate(block.Bytes)
					Expect(err).To(Succeed())
					originalCANotAfter = ca.NotAfter
				})

				By("Patching the apiserver.crt to expire in 30 days", func() {
					secret, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
						certs.CertSecretName(vClusterName), metav1.GetOptions{})
					Expect(err).To(Succeed())

					expiringCert := generateExpiringCertPEM(30 * 24 * time.Hour)
					secret.Data["apiserver.crt"] = expiringCert

					_, err = hostClient.CoreV1().Secrets(vClusterNamespace).Update(ctx,
						secret, metav1.UpdateOptions{})
					Expect(err).To(Succeed())
				})
			})

			// Spec 3 depends on 2: delete the pod so it restarts and triggers auto-rotation
			It("should restart the vCluster pod to trigger auto-rotation", func(ctx context.Context) {
				By("Deleting the vCluster pod", func() {
					pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
						LabelSelector: "app=vcluster,release=" + vClusterName,
					})
					Expect(err).To(Succeed())
					Expect(pods.Items).NotTo(BeEmpty())

					for _, pod := range pods.Items {
						err := hostClient.CoreV1().Pods(vClusterNamespace).Delete(ctx,
							pod.Name, metav1.DeleteOptions{})
						Expect(err).To(Succeed())
					}
				})

				By("Waiting for the pod to come back ready", func() {
					waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutVeryLong)
				})
			})

			// Spec 4 depends on 3: verify the cert was auto-rotated.
			It("should have a renewed apiserver cert after auto-rotation", func(ctx context.Context) {
				expectCertRenewed(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutVeryLong)
			})

			// Spec 5 depends on 3: verify CA was NOT rotated
			It("should preserve the CA cert during auto-rotation", func(ctx context.Context) {
				secret, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
					certs.CertSecretName(vClusterName), metav1.GetOptions{})
				Expect(err).To(Succeed())

				block, _ := pem.Decode(secret.Data["ca.crt"])
				Expect(block).NotTo(BeNil(), "failed to decode CA cert PEM")

				ca, err := x509.ParseCertificate(block.Bytes)
				Expect(err).To(Succeed())

				Expect(ca.NotAfter.Equal(originalCANotAfter)).To(BeTrue(),
					"CA cert should not have been rotated, original NotAfter=%s, current NotAfter=%s",
					originalCANotAfter.Format(time.RFC3339), ca.NotAfter.Format(time.RFC3339))
			})

			// Spec 6: verify all leaf certs are valid
			It("should have all leaf certs valid after auto-rotation", func(ctx context.Context) {
				secret, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
					certs.CertSecretName(vClusterName), metav1.GetOptions{})
				Expect(err).To(Succeed())

				leafCerts := []string{
					"apiserver.crt",
					"apiserver-kubelet-client.crt",
					"apiserver-etcd-client.crt",
					"front-proxy-client.crt",
					"etcd-server.crt",
					"etcd-peer.crt",
					"etcd-healthcheck-client.crt",
				}

				for _, certName := range leafCerts {
					data, ok := secret.Data[certName]
					Expect(ok).To(BeTrue(), "cert %s should exist in secret", certName)

					block, _ := pem.Decode(data)
					Expect(block).NotTo(BeNil(), "failed to decode %s PEM", certName)

					cert, err := x509.ParseCertificate(block.Bytes)
					Expect(err).To(Succeed(), "failed to parse %s", certName)

					Expect(cert.NotAfter.After(time.Now().Add(90*24*time.Hour))).To(BeTrue(),
						"%s should be valid for more than 90 days, NotAfter=%s", certName, cert.NotAfter.Format(time.RFC3339))
				}
			})
		},
	)
}

// generateExpiringCertPEM creates a self-signed certificate that expires at
// the given duration from now.
func generateExpiringCertPEM(expiresIn time.Duration) []byte {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "kube-apiserver"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(expiresIn),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})
}
