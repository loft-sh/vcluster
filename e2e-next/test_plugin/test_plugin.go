package test_plugin

import (
	"context"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var _ = Describe("Plugin",
	labels.Plugin,
	labels.PR,
	cluster.Use(clusters.PluginVCluster),
	cluster.Use(clusters.HostCluster),
	func() {
		var (
			hostClient        kubernetes.Interface
			vClusterClient    kubernetes.Interface
			vClusterName      = clusters.PluginVClusterName
			vClusterNamespace = "vcluster-" + clusters.PluginVClusterName
		)

		BeforeEach(func(ctx context.Context) {
			hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
			Expect(hostClient).NotTo(BeNil())
			vClusterClient = cluster.CurrentKubeClientFrom(ctx)
			Expect(vClusterClient).NotTo(BeNil())
		})

		It("verifies bootstrap plugin deploys a Deployment into the vcluster", func(ctx context.Context) {
			By("waiting for the bootstrap plugin to create the deployment", func() {
				Eventually(func(g Gomega) {
					_, err := vClusterClient.AppsV1().Deployments("default").Get(ctx, "mydeployment", metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred(), "bootstrap plugin should create 'mydeployment' in default namespace")
				}).
					WithPolling(constants.PollingInterval).
					WithTimeout(constants.PollingTimeout).
					Should(Succeed())
			})
		})

		It("verifies hooks plugin adds a port to synced Services", func(ctx context.Context) {
			svcName := fmt.Sprintf("plugin-hooks-test-%d", GinkgoRandomSeed())
			translatedName := translate.SingleNamespaceHostName(svcName, "default", vClusterName)

			By("creating a ClusterIP Service in the vcluster", func() {
				_, err := vClusterClient.CoreV1().Services("default").Create(ctx, &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      svcName,
						Namespace: "default",
					},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
						Ports: []corev1.ServicePort{
							{
								Name: "test",
								Port: 1000,
							},
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Services("default").Delete(ctx, svcName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).NotTo(HaveOccurred())
					}
				})
			})

			By("waiting for the synced host Service to appear with the injected plugin port", func() {
				Eventually(func(g Gomega) {
					hostSvc, err := hostClient.CoreV1().Services(vClusterNamespace).Get(ctx, translatedName, metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred(), "synced service should exist on host")
					g.Expect(hostSvc.Spec.Ports).To(HaveLen(2),
						"hooks plugin should add a second port, got ports: %v", hostSvc.Spec.Ports)
					g.Expect(hostSvc.Spec.Ports[1].Name).To(Equal("plugin"),
						"second port should be named 'plugin'")
					g.Expect(hostSvc.Spec.Ports[1].Port).To(Equal(int32(19000)),
						"plugin port should be 19000")
				}).
					WithPolling(constants.PollingInterval).
					WithTimeout(constants.PollingTimeout).
					Should(Succeed())
			})
		})

		It("verifies import-secrets plugin imports and syncs host Secrets into vcluster", func(ctx context.Context) {
			hostSecretName := fmt.Sprintf("plugin-import-test-%d", GinkgoRandomSeed())
			const (
				importTargetNS   = "test"
				importTargetName = "test"
			)

			By("creating a host Secret with the import annotation", func() {
				_, err := hostClient.CoreV1().Secrets(vClusterNamespace).Create(ctx, &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      hostSecretName,
						Namespace: vClusterNamespace,
						Annotations: map[string]string{
							"vcluster.loft.sh/import": importTargetNS + "/" + importTargetName,
						},
					},
					Data: map[string][]byte{
						"test": []byte("test"),
					},
				}, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				DeferCleanup(func(ctx context.Context) {
					err := hostClient.CoreV1().Secrets(vClusterNamespace).Delete(ctx, hostSecretName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).NotTo(HaveOccurred())
					}
				})
			})

			By("waiting for the imported Secret to appear in the vcluster", func() {
				Eventually(func(g Gomega) {
					vSecret, err := vClusterClient.CoreV1().Secrets(importTargetNS).Get(ctx, importTargetName, metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred(), "imported secret should exist in vcluster at %s/%s", importTargetNS, importTargetName)
					g.Expect(vSecret.Data).To(HaveLen(1), "imported secret should have 1 data key")
					g.Expect(string(vSecret.Data["test"])).To(Equal("test"), "imported secret data should match")
				}).
					WithPolling(constants.PollingInterval).
					WithTimeout(constants.PollingTimeout).
					Should(Succeed())
			})

			By("updating the host Secret and verifying the change propagates", func() {
				Eventually(func(g Gomega) {
					secret, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx, hostSecretName, metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred())
					secret.Data["test"] = []byte("newtest")
					_, err = hostClient.CoreV1().Secrets(vClusterNamespace).Update(ctx, secret, metav1.UpdateOptions{})
					g.Expect(err).NotTo(HaveOccurred())
				}).
					WithPolling(constants.PollingInterval).
					WithTimeout(constants.PollingTimeoutShort).
					Should(Succeed())

				Eventually(func(g Gomega) {
					vSecret, err := vClusterClient.CoreV1().Secrets(importTargetNS).Get(ctx, importTargetName, metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(string(vSecret.Data["test"])).To(Equal("newtest"), "imported secret should reflect updated value")
				}).
					WithPolling(constants.PollingInterval).
					WithTimeout(constants.PollingTimeout).
					Should(Succeed())
			})

			By("deleting the host Secret and verifying the vcluster Secret is removed", func() {
				Expect(hostClient.CoreV1().Secrets(vClusterNamespace).Delete(ctx, hostSecretName, metav1.DeleteOptions{})).To(Succeed())

				Eventually(func(g Gomega) {
					_, err := vClusterClient.CoreV1().Secrets(importTargetNS).Get(ctx, importTargetName, metav1.GetOptions{})
					g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "imported secret should be deleted after host secret removal")
				}).
					WithPolling(constants.PollingInterval).
					WithTimeout(constants.PollingTimeout).
					Should(Succeed())
			})
		})
	},
)
