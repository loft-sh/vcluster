// Package plugin contains legacy vCluster plugin tests (v1 and v2).
package plugin

import (
	"context"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// PluginSpec registers legacy plugin tests (bootstrap-with-deployment, hooks, import-secrets).
func PluginSpec() {
	Describe("Legacy plugin tests",
		labels.PR, labels.Integration, labels.Plugin,
		func() {
			var (
				vClusterClient    kubernetes.Interface
				hostClient        kubernetes.Interface
				vClusterName      string
				vClusterNamespace string
			)

			BeforeEach(func(ctx context.Context) context.Context {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterNamespace = "vcluster-" + vClusterName
				return ctx
			})

			It("should create a deployment via the legacy bootstrap-with-deployment plugin", func(ctx context.Context) {
				By("waiting for the bootstrap-with-deployment plugin to create 'mydeployment' in default namespace", func() {
					Eventually(func(g Gomega) {
						dep, err := vClusterClient.AppsV1().Deployments("default").Get(ctx, "mydeployment", metav1.GetOptions{})
						g.Expect(err).To(Succeed(), "mydeployment not yet created by bootstrap-with-deployment plugin")
						g.Expect(dep.Name).To(Equal("mydeployment"))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("should inject an additional port via the hooks plugin", func(ctx context.Context) {
				suffix := random.String(6)
				svcName := "hooks-test-" + suffix

				By("creating a service in the virtual cluster", func() {
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
					Expect(err).To(Succeed())
				})
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Services("default").Delete(ctx, svcName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				hostSvcName := translate.SingleNamespaceHostName(svcName, "default", vClusterName)

				By("waiting for the service to be synced to the host with the injected plugin port", func() {
					Eventually(func(g Gomega) {
						hostSvc, err := hostClient.CoreV1().Services(vClusterNamespace).Get(ctx, hostSvcName, metav1.GetOptions{})
						g.Expect(err).To(Succeed(),
							"host service %s/%s not yet synced", vClusterNamespace, hostSvcName)
						g.Expect(hostSvc.Spec.Ports).To(HaveLen(2),
							"expected 2 ports (original + plugin), got %d: %v", len(hostSvc.Spec.Ports), hostSvc.Spec.Ports)
						g.Expect(hostSvc.Spec.Ports[1].Name).To(Equal("plugin"),
							"second port should be named 'plugin'")
						g.Expect(hostSvc.Spec.Ports[1].Port).To(Equal(int32(19000)),
							"plugin port should be 19000")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("should import, update, and delete a secret via the import-secrets plugin", func(ctx context.Context) {
				suffix := random.String(6)
				hostSecretName := "import-test-" + suffix
				vSecretName := "imported-" + suffix
				vSecretNS := "imported-ns-" + suffix

				By("creating a namespace in the virtual cluster for the imported secret", func() {
					_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: vSecretNS},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Namespaces().Delete(ctx, vSecretNS, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				By("creating a host secret with the import annotation", func() {
					_, err := hostClient.CoreV1().Secrets(vClusterNamespace).Create(ctx, &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      hostSecretName,
							Namespace: vClusterNamespace,
							Annotations: map[string]string{
								"vcluster.loft.sh/import": fmt.Sprintf("%s/%s", vSecretNS, vSecretName),
							},
						},
						Data: map[string][]byte{
							"key": []byte("value"),
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})
				DeferCleanup(func(ctx context.Context) {
					err := hostClient.CoreV1().Secrets(vClusterNamespace).Delete(ctx, hostSecretName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				By("waiting for the secret to appear in the virtual cluster", func() {
					Eventually(func(g Gomega) {
						vSecret, err := vClusterClient.CoreV1().Secrets(vSecretNS).Get(ctx, vSecretName, metav1.GetOptions{})
						g.Expect(err).To(Succeed(),
							"secret %s/%s not yet imported into vcluster", vSecretNS, vSecretName)
						g.Expect(vSecret.Data).To(HaveKeyWithValue("key", []byte("value")),
							"imported secret should have key=value")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				By("updating the host secret and verifying the change propagates", func() {
					hostSecret, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx, hostSecretName, metav1.GetOptions{})
					Expect(err).To(Succeed())
					hostSecret.Data["key"] = []byte("newvalue")
					_, err = hostClient.CoreV1().Secrets(vClusterNamespace).Update(ctx, hostSecret, metav1.UpdateOptions{})
					Expect(err).To(Succeed())

					Eventually(func(g Gomega) {
						vSecret, err := vClusterClient.CoreV1().Secrets(vSecretNS).Get(ctx, vSecretName, metav1.GetOptions{})
						g.Expect(err).To(Succeed(),
							"secret %s/%s disappeared after host update", vSecretNS, vSecretName)
						g.Expect(vSecret.Data).To(HaveKeyWithValue("key", []byte("newvalue")),
							"imported secret should reflect updated value")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				By("deleting the host secret and verifying the vcluster secret is removed", func() {
					err := hostClient.CoreV1().Secrets(vClusterNamespace).Delete(ctx, hostSecretName, metav1.DeleteOptions{})
					Expect(err).To(Succeed())

					Eventually(func(g Gomega) {
						_, err := vClusterClient.CoreV1().Secrets(vSecretNS).Get(ctx, vSecretName, metav1.GetOptions{})
						g.Expect(kerrors.IsNotFound(err)).To(BeTrue(),
							"secret %s/%s should be deleted after host secret removal, got err: %v", vSecretNS, vSecretName, err)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})
		},
	)
}
