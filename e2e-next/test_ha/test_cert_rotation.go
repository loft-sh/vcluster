package test_ha

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"time"

	certscmd "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/certs"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/cli/flags"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var haVClusterNamespace = "vcluster-" + clusters.HAVClusterName

// Ordered: CA rotation must complete before leaf rotation because both restart
// the vCluster. The leaf rotation spec's precondition is a recovered vCluster
// with a fresh CA — the side effect of the CA rotation spec.
var _ = Describe("Cert rotation on HA vCluster", Ordered, labels.Core, labels.PR,
	cluster.Use(clusters.HAVCluster), cluster.Use(clusters.HostCluster),
	func() {
		var (
			hostClient kubernetes.Interface
		)

		BeforeAll(func(ctx context.Context) {
			By("Obtaining host client", func() {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
			})

			By("Verifying all HA vCluster replicas are ready", func() {
				waitForDeploymentReady(ctx, hostClient, haVClusterNamespace, clusters.HAVClusterName)
			})
		})

		It("should rotate the CA certificate and recover", func(ctx context.Context) {
			By("Rotating the CA certificate", func() {
				certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: haVClusterNamespace})
				certsCmd.SetArgs([]string{"rotate-ca", clusters.HAVClusterName})
				Expect(certsCmd.Execute()).To(Succeed())
			})

			By("Waiting for the HA vCluster to become ready", func() {
				waitForDeploymentReady(ctx, hostClient, haVClusterNamespace, clusters.HAVClusterName)
			})

			By("Verifying the new CA certificate is valid", func() {
				certData := parseCertFromSecret(ctx, hostClient, haVClusterNamespace, clusters.HAVClusterName, "ca.crt")
				Expect(certData.NotAfter.After(time.Now())).To(BeTrue(),
					"CA cert NotAfter (%s) should be in the future", certData.NotAfter)
			})
		})

		It("should rotate the leaf certificate and recover", func(ctx context.Context) {
			By("Rotating the leaf certificate", func() {
				certsCmd := certscmd.NewCertsCmd(&flags.GlobalFlags{Namespace: haVClusterNamespace})
				certsCmd.SetArgs([]string{"rotate", clusters.HAVClusterName})
				Expect(certsCmd.Execute()).To(Succeed())
			})

			By("Waiting for the HA vCluster to become ready", func() {
				waitForDeploymentReady(ctx, hostClient, haVClusterNamespace, clusters.HAVClusterName)
			})

			By("Verifying the new leaf certificate is valid", func() {
				certData := parseCertFromSecret(ctx, hostClient, haVClusterNamespace, clusters.HAVClusterName, "apiserver-etcd-client.crt")
				Expect(certData.NotAfter.After(time.Now())).To(BeTrue(),
					"Leaf cert NotAfter (%s) should be in the future", certData.NotAfter)
			})
		})
	},
)

// waitForDeploymentReady polls until the Deployment for the vCluster has all replicas ready.
func waitForDeploymentReady(ctx context.Context, hostClient kubernetes.Interface, namespace, deploymentName string) {
	Eventually(func(g Gomega) {
		deploy, err := hostClient.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
		g.Expect(err).NotTo(HaveOccurred(), "failed to get Deployment %s/%s", namespace, deploymentName)
		g.Expect(deploy.Spec.Replicas).NotTo(BeNil(), "Deployment %s/%s has nil spec.replicas", namespace, deploymentName)
		g.Expect(deploy.Status.ReadyReplicas).To(Equal(*deploy.Spec.Replicas),
			"Deployment %s/%s: ready=%d, want=%d",
			namespace, deploymentName, deploy.Status.ReadyReplicas, *deploy.Spec.Replicas)
	}).WithPolling(constants.PollingInterval).
		WithTimeout(constants.PollingTimeoutVeryLong).
		Should(Succeed())
}

// parseCertFromSecret fetches the cert secret and parses the specified PEM-encoded certificate field.
func parseCertFromSecret(ctx context.Context, hostClient kubernetes.Interface, namespace, vclusterName, certKey string) *x509.Certificate {
	secret, err := hostClient.CoreV1().Secrets(namespace).Get(
		ctx, certs.CertSecretName(vclusterName), metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "failed to get cert secret for %s/%s", namespace, vclusterName)

	certPEM, ok := secret.Data[certKey]
	Expect(ok).To(BeTrue(), "cert secret should contain key %q", certKey)

	block, _ := pem.Decode(certPEM)
	Expect(block).NotTo(BeNil(), "failed to decode PEM block from %q", certKey)

	cert, err := x509.ParseCertificate(block.Bytes)
	Expect(err).NotTo(HaveOccurred(), "failed to parse certificate from %q", certKey)

	return cert
}
