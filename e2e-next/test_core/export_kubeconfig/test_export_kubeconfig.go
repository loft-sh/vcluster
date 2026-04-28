package export_kubeconfig

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func ExportKubeConfigSpec() {
	Describe("export kubeconfig",
		labels.Core,
		func() {
			var (
				hostClient     kubernetes.Interface
				vClusterName   string
				vClusterHostNS string
			)

			BeforeEach(func(ctx context.Context) context.Context {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterHostNS = "vcluster-" + vClusterName
				return ctx
			})

			It("preserves the default kubeconfig secret", func(ctx context.Context) {
				defaultSecretName := "vc-" + vClusterName

				By("waiting for the default kubeconfig secret to exist", func() {
					Eventually(func(g Gomega) {
						secret, err := hostClient.CoreV1().Secrets(vClusterHostNS).Get(ctx, defaultSecretName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "default kubeconfig secret should exist")

						g.Expect(secret.Data["config"]).NotTo(BeEmpty(), "kubeconfig data should not be empty")

						kubeConfig, err := clientcmd.Load(secret.Data["config"])
						g.Expect(err).NotTo(HaveOccurred(), "kubeconfig should be parseable")

						g.Expect(kubeConfig.Clusters).To(HaveLen(1), "should have exactly one cluster entry")
						for _, clusterEntry := range kubeConfig.Clusters {
							g.Expect(clusterEntry.Server).To(Equal("https://localhost:8443"),
								"default secret should use localhost:8443")
						}

						g.Expect(secret.Labels).To(HaveKeyWithValue("app", "vcluster"),
							"secret should have app=vcluster label")
						g.Expect(secret.Labels).To(HaveKeyWithValue("vcluster-name", vClusterName),
							"secret should have vcluster-name label")
					}).WithPolling(constants.PollingInterval).
						WithTimeout(constants.PollingTimeout).
						Should(Succeed())
				})
			})

			It("creates same-namespace additional secret with overrides", func(ctx context.Context) {
				By("waiting for the same-namespace additional secret to exist", func() {
					Eventually(func(g Gomega) {
						secret, err := hostClient.CoreV1().Secrets(vClusterHostNS).Get(
							ctx, SameNSSecretName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(),
							"same-namespace additional secret should exist in %s", vClusterHostNS)

						g.Expect(secret.Data["config"]).NotTo(BeEmpty(), "kubeconfig data should not be empty")

						kubeConfig, err := clientcmd.Load(secret.Data["config"])
						g.Expect(err).NotTo(HaveOccurred(), "kubeconfig should be parseable")

						g.Expect(kubeConfig.Clusters).To(HaveLen(1), "should have exactly one cluster entry")
						for _, clusterEntry := range kubeConfig.Clusters {
							g.Expect(clusterEntry.Server).To(Equal(SameNSServer),
								"server should match configured override")
						}

						g.Expect(kubeConfig.Contexts).To(HaveKey(SameNSContext),
							"context name should match configured override")

						g.Expect(secret.Data["certificate-authority"]).NotTo(BeEmpty(),
							"CA data should be present")

						g.Expect(secret.Labels).To(HaveKeyWithValue("app", "vcluster"),
							"secret should have app=vcluster label")
						g.Expect(secret.Labels).To(HaveKeyWithValue("vcluster-name", vClusterName),
							"secret should have vcluster-name label")
					}).WithPolling(constants.PollingInterval).
						WithTimeout(constants.PollingTimeout).
						Should(Succeed())
				})
			})

			It("creates cross-namespace additional secret", func(ctx context.Context) {
				By("waiting for the cross-namespace additional secret to exist", func() {
					Eventually(func(g Gomega) {
						secret, err := hostClient.CoreV1().Secrets(TargetNS).Get(
							ctx, CrossNSSecretName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(),
							"cross-namespace additional secret should exist in %s", TargetNS)

						g.Expect(secret.Data["config"]).NotTo(BeEmpty(), "kubeconfig data should not be empty")

						kubeConfig, err := clientcmd.Load(secret.Data["config"])
						g.Expect(err).NotTo(HaveOccurred(), "kubeconfig should be parseable")

						g.Expect(kubeConfig.Clusters).To(HaveLen(1), "should have exactly one cluster entry")
						for _, clusterEntry := range kubeConfig.Clusters {
							g.Expect(clusterEntry.Server).To(Equal(CrossNSServer),
								"server should match configured override")
						}

						g.Expect(kubeConfig.Contexts).To(HaveKey(CrossNSContext),
							"context name should match configured override")

						g.Expect(secret.Labels).To(HaveKeyWithValue("app", "vcluster"),
							"secret should have app=vcluster label")
						g.Expect(secret.Labels).To(HaveKeyWithValue("vcluster-name", vClusterName),
							"secret should have vcluster-name label")
					}).WithPolling(constants.PollingInterval).
						WithTimeout(constants.PollingTimeout).
						Should(Succeed())
				})
			})

			It("exports valid kubeconfig credentials", func(ctx context.Context) {
				By("verifying the kubeconfig from the same-namespace secret produces a valid REST config", func() {
					Eventually(func(g Gomega) {
						secret, err := hostClient.CoreV1().Secrets(vClusterHostNS).Get(
							ctx, SameNSSecretName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "additional secret should exist")
						g.Expect(secret.Data["config"]).NotTo(BeEmpty(), "kubeconfig data should not be empty")

						restConfig, err := clientcmd.RESTConfigFromKubeConfig(secret.Data["config"])
						g.Expect(err).NotTo(HaveOccurred(), "should produce a valid REST config")
						g.Expect(restConfig.TLSClientConfig.CertData).NotTo(BeEmpty(),
							"client certificate should be present")
						g.Expect(restConfig.TLSClientConfig.KeyData).NotTo(BeEmpty(),
							"client key should be present")
					}).WithPolling(constants.PollingInterval).
						WithTimeout(constants.PollingTimeout).
						Should(Succeed())
				})

				By("connecting to the vCluster using the exported credentials", func() {
					// Build a client from the additional secret's credentials (cert, key, CA)
					// but route through the framework's background proxy since the configured
					// server URL is in-cluster DNS not reachable from the test runner.
					Eventually(func(g Gomega) {
						secret, err := hostClient.CoreV1().Secrets(vClusterHostNS).Get(
							ctx, SameNSSecretName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "additional secret should exist")

						restConfig, err := clientcmd.RESTConfigFromKubeConfig(secret.Data["config"])
						g.Expect(err).NotTo(HaveOccurred(), "should parse REST config from exported secret")

						// The exported server URL points to in-cluster DNS which isn't routable
						// from the test runner. Route through the background proxy instead — this
						// still authenticates with the exported secret's TLS credentials.
						proxyHost := cluster.CurrentClusterFrom(ctx).KubernetesRestConfig().Host
						restConfig.Host = proxyHost

						client, err := kubernetes.NewForConfig(restConfig)
						g.Expect(err).NotTo(HaveOccurred(), "should create kubernetes client")

						version, err := client.Discovery().ServerVersion()
						g.Expect(err).NotTo(HaveOccurred(), "exported credentials should authenticate against the vCluster API")
						g.Expect(version.GitVersion).NotTo(BeEmpty(), "server version should be non-empty")
					}).WithPolling(constants.PollingInterval).
						WithTimeout(constants.PollingTimeout).
						Should(Succeed())
				})
			})
		},
	)
}
