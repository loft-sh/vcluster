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
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/certs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DescribeCertAutoRotation registers tests that verify automatic certificate
// rotation when leaf certs are near expiry. These tests are Ordered because
// they form a lifecycle: record state -> inject expiring cert -> restart -> verify.
func DescribeCertAutoRotation(vcluster suite.Dependency) bool {
	return Describe("vCluster cert auto-rotation",
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

			// Spec 1: verify vCluster is ready before we start
			It("should have all vCluster pods running and ready", func(ctx context.Context) {
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
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
			})

			// Spec 2 depends on 1: record original cert NotAfter and inject an expiring cert
			var originalCANotAfter time.Time

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

			// Spec 3 depends on 2: delete the pod so it restarts and picks up the expiring cert
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
					Eventually(func(g Gomega) {
						pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
							LabelSelector: "app=vcluster,release=" + vClusterName,
						})
						g.Expect(err).To(Succeed())
						g.Expect(pods.Items).NotTo(BeEmpty())
						for _, pod := range pods.Items {
							g.Expect(pod.Status.ContainerStatuses).NotTo(BeEmpty())
							for _, container := range pod.Status.ContainerStatuses {
								g.Expect(container.State.Running).NotTo(BeNil(),
									"container %s in pod %s not running after restart", container.Name, pod.Name)
								g.Expect(container.Ready).To(BeTrue(),
									"container %s in pod %s not ready after restart", container.Name, pod.Name)
							}
						}
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
				})
			})

			// Spec 4 depends on 3: verify the cert was auto-rotated
			It("should have a renewed apiserver cert after auto-rotation", func(ctx context.Context) {
				secret, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
					certs.CertSecretName(vClusterName), metav1.GetOptions{})
				Expect(err).To(Succeed())

				block, _ := pem.Decode(secret.Data["apiserver.crt"])
				Expect(block).NotTo(BeNil(), "failed to decode apiserver cert PEM")

				cert, err := x509.ParseCertificate(block.Bytes)
				Expect(err).To(Succeed())

				// The renewed cert should expire more than 90 days from now
				// (it should be ~365 days since it was just regenerated)
				Expect(cert.NotAfter.After(time.Now().Add(90 * 24 * time.Hour))).To(BeTrue(),
					"apiserver cert should have been renewed, NotAfter=%s", cert.NotAfter.Format(time.RFC3339))
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

			// Spec 6: verify all certs are valid
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

					Expect(cert.NotAfter.After(time.Now().Add(90 * 24 * time.Hour))).To(BeTrue(),
						"%s should be valid for more than 90 days, NotAfter=%s", certName, cert.NotAfter.Format(time.RFC3339))
				}
			})
		},
	)
}

// generateExpiringCertPEM creates a self-signed certificate that expires at
// the given duration from now. Used to inject expiring certs into the secret.
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
