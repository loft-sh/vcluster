package test_gatewayapi

import (
	"context"

	"github.com/loft-sh/vcluster/e2e/constants"
	"github.com/loft-sh/vcluster/e2e/labels"
	"github.com/loft-sh/vcluster/pkg/controllers/resources/gateways"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const combinedClassSelectorValue = "gatewayapi-combined"

// GatewayAPICombinedSpec covers ENGNODE-556 / TC-33: importing host Gateways
// and syncing tenant Gateways can be enabled together without startup loops.
func GatewayAPICombinedSpec() {
	Describe("Gateway API combined import and tenant Gateway sync", labels.GatewayAPI, labels.GatewayClasses, func() {
		var (
			hostClient     ctrlclient.Client
			vClusterClient ctrlclient.Client
			vClusterName   string
			vClusterHostNS string
		)

		gatewayNamespace := "gwapi-combined-host"
		BeforeEach(func(ctx context.Context) {
			clients := newGatewayAPIClients(ctx)
			hostClient = clients.HostClient
			vClusterClient = clients.VClusterClient
			vClusterName = clients.VClusterName
			vClusterHostNS = clients.VClusterHostNS

			ensureHostNamespace(ctx, hostClient, gatewayNamespace)
			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.List(ctx, &corev1.NamespaceList{})).To(Succeed())
				g.Expect(vClusterClient.List(ctx, &gatewayv1.GatewayClassList{})).To(Succeed())
				g.Expect(vClusterClient.List(ctx, &gatewayv1.GatewayList{}, ctrlclient.InNamespace("default"))).To(Succeed())
				g.Expect(vClusterClient.List(ctx, &gatewayv1.HTTPRouteList{}, ctrlclient.InNamespace("default"))).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
		})

		It("starts with both Gateway import and tenant Gateway sync enabled", func(ctx context.Context) {
			suffix := random.String(6)
			class := createGatewayClass(ctx, hostClient, "gwc-combined-"+suffix, combinedClassSelectorValue, "combined class")
			hostGW := hostGateway(gatewayNamespace, "edge-"+suffix, class.Name)
			createHostGateway(ctx, hostClient, hostGW)

			importedGatewayNamespace := "gwapi-combined-import"
			By("verifying the imported Gateway mirror appears after vCluster startup", func() {
				Eventually(func(g Gomega) {
					mirror := &gatewayv1.Gateway{}
					g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Namespace: importedGatewayNamespace, Name: hostGW.Name}, mirror)).To(Succeed())
					g.Expect(mirror.Labels).To(HaveKeyWithValue(gateways.ImportedGatewayLabel, "true"))
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})

			By("verifying the combined vCluster syncs tenant-authored Gateway API resources to host", func() {
				Eventually(func(g Gomega) {
					g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Name: class.Name}, &gatewayv1.GatewayClass{})).To(Succeed())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

				tenantNS := createTenantNamespace(ctx, vClusterClient, "gwapi-combined-tenant-"+suffix)
				service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "backend-" + suffix, Namespace: tenantNS.Name}, Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}}}
				Expect(vClusterClient.Create(ctx, service)).To(Succeed())

				tenantGW := tenantGateway(tenantNS.Name, "tenant-gw-"+suffix, class.Name)
				Expect(vClusterClient.Create(ctx, tenantGW)).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, tenantGW))).To(Succeed())
				})
				tenantRoute := tenantHTTPRoute(tenantNS.Name, "tenant-route-"+suffix, tenantGW.Name, service.Name)
				Expect(vClusterClient.Create(ctx, tenantRoute)).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, tenantRoute))).To(Succeed())
				})

				hostGatewayName := translate.SafeConcatName(tenantGW.Name, "x", tenantNS.Name, "x", vClusterName)
				hostRouteName := translate.SafeConcatName(tenantRoute.Name, "x", tenantNS.Name, "x", vClusterName)
				Eventually(func(g Gomega) {
					gotGateway := &gatewayv1.Gateway{}
					g.Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostGatewayName}, gotGateway)).To(Succeed())
					g.Expect(gotGateway.Spec.GatewayClassName).To(Equal(gatewayv1.ObjectName(class.Name)))

					gotRoute := &gatewayv1.HTTPRoute{}
					g.Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostRouteName}, gotRoute)).To(Succeed())
					g.Expect(gotRoute.Spec.ParentRefs).To(HaveLen(1))
					g.Expect(gotRoute.Spec.ParentRefs[0].Name).To(Equal(gatewayv1.ObjectName(hostGatewayName)))
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

				Consistently(func(g Gomega) {
					g.Expect(vClusterClient.Get(ctx, ctrlclient.ObjectKeyFromObject(tenantGW), &gatewayv1.Gateway{})).To(Succeed())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
			})
		})
	})
}
