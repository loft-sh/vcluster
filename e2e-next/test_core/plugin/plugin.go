package plugin

import (
	"context"

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

// PluginSpec registers specs for legacy vCluster plugins (v1 and v2).
func PluginSpec() {
	Describe("Plugin",
		labels.Integration,
		func() {
			var (
				hostClient     kubernetes.Interface
				vClusterClient kubernetes.Interface
				vClusterName   string
				hostNS         string
			)

			BeforeEach(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				hostNS = "vcluster-" + vClusterName
			})

			It("should create a deployment via the legacy bootstrap-with-deployment plugin", func(ctx context.Context) {
				By("waiting for the plugin-bootstrapped deployment to appear in vCluster", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterClient.AppsV1().Deployments("default").Get(ctx, "mydeployment", metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "deployment mydeployment not yet available in vCluster")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})

			It("should inject an additional port via the hooks plugin", func(ctx context.Context) {
				suffix := random.String(6)
				svcName := "plugin-hooks-svc-" + suffix

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
								Port: int32(1000),
							},
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Services("default").Delete(ctx, svcName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				hostSvcName := translate.SingleNamespaceHostName(svcName, "default", vClusterName)

				var hostService *corev1.Service
				By("waiting for the service to be synced to the host cluster", func() {
					Eventually(func(g Gomega) {
						var err error
						hostService, err = hostClient.CoreV1().Services(hostNS).Get(ctx, hostSvcName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "host service %s/%s not yet available", hostNS, hostSvcName)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("verifying the hooks plugin injected an additional port on the host service", func() {
					Expect(hostService.Spec.Ports).To(HaveLen(2),
						"expected 2 ports on host service %s/%s (original + plugin-injected), got: %v",
						hostNS, hostSvcName, hostService.Spec.Ports)
					Expect(hostService.Spec.Ports[1].Name).To(Equal("plugin"),
						"expected plugin port name on host service %s/%s", hostNS, hostSvcName)
					Expect(hostService.Spec.Ports[1].Port).To(Equal(int32(19000)),
						"expected plugin port 19000 on host service %s/%s", hostNS, hostSvcName)
				})
			})

			It("should import, update, and delete a secret via the import-secrets plugin", func(ctx context.Context) {
				suffix := random.String(6)
				hostSecretName := "import-secret-test-" + suffix
				vSecretNS := "test-" + suffix
				vSecretName := "test-" + suffix

				_, err := hostClient.CoreV1().Secrets(hostNS).Create(ctx, &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      hostSecretName,
						Namespace: hostNS,
						Annotations: map[string]string{
							"vcluster.loft.sh/import": vSecretNS + "/" + vSecretName,
						},
					},
					Data: map[string][]byte{
						"test": []byte("test"),
					},
				}, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					err := hostClient.CoreV1().Secrets(hostNS).Delete(ctx, hostSecretName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				By("waiting for the imported secret to appear in vCluster", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterClient.CoreV1().Secrets(vSecretNS).Get(ctx, vSecretName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "imported secret %s/%s not yet available in vCluster", vSecretNS, vSecretName)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("verifying the imported secret data", func() {
					vSecret, err := vClusterClient.CoreV1().Secrets(vSecretNS).Get(ctx, vSecretName, metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(vSecret.Data).To(HaveLen(1),
						"expected exactly 1 data key in imported secret, got: %v", vSecret.Data)
					Expect(vSecret.Data).To(HaveKeyWithValue("test", []byte("test")),
						"expected secret data key 'test' with value 'test'")
				})

				By("updating the host secret data", func() {
					hostSecret, err := hostClient.CoreV1().Secrets(hostNS).Get(ctx, hostSecretName, metav1.GetOptions{})
					Expect(err).To(Succeed())
					hostSecret.Data["test"] = []byte("newtest")
					_, err = hostClient.CoreV1().Secrets(hostNS).Update(ctx, hostSecret, metav1.UpdateOptions{})
					Expect(err).To(Succeed())
				})

				By("waiting for the updated data to propagate to the vCluster secret", func() {
					Eventually(func(g Gomega) {
						vSecret, err := vClusterClient.CoreV1().Secrets(vSecretNS).Get(ctx, vSecretName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "vCluster secret %s/%s not yet available", vSecretNS, vSecretName)
						g.Expect(vSecret.Data).To(HaveKeyWithValue("test", []byte("newtest")),
							"expected updated value 'newtest' in vCluster secret %s/%s, got data: %v", vSecretNS, vSecretName, vSecret.Data)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("deleting the host secret", func() {
					err := hostClient.CoreV1().Secrets(hostNS).Delete(ctx, hostSecretName, metav1.DeleteOptions{})
					Expect(err).To(Succeed())
				})

				By("waiting for the imported secret to be removed from vCluster", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterClient.CoreV1().Secrets(vSecretNS).Get(ctx, vSecretName, metav1.GetOptions{})
						g.Expect(kerrors.IsNotFound(err)).To(BeTrue(),
							"expected secret %s/%s to be deleted from vCluster, got err: %v", vSecretNS, vSecretName, err)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})
		},
	)
}
