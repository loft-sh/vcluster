package test_gatewayapi

import (
	"context"

	"github.com/loft-sh/vcluster/e2e/constants"
	"github.com/loft-sh/vcluster/e2e/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const umbrellaClassSelectorValue = "gatewayapi-umbrella"

// GatewayAPIUmbrellaSpec registers coverage for the sync.toHost.gatewayApi
// umbrella switch: with only gatewayApi.enabled set, the tenant cluster must
// serve the Gateway, GatewayClass, HTTPRoute and ReferenceGrant CRDs and sync
// tenant Gateways and HTTPRoutes to the host. ReferenceGrants gate
// cross-namespace routes through tenant-side validation only; without
// namespace sync they are never written to the host.
func GatewayAPIUmbrellaSpec() {
	Describe("Gateway API umbrella switch", labels.GatewayAPI, func() {
		var (
			hostClient     ctrlclient.Client
			vClusterClient ctrlclient.Client
			vClusterName   string
			vClusterHostNS string
		)

		BeforeEach(func(ctx context.Context) {
			clients := newGatewayAPIClients(ctx)
			hostClient = clients.HostClient
			vClusterClient = clients.VClusterClient
			vClusterName = clients.VClusterName
			vClusterHostNS = clients.VClusterHostNS
		})

		It("serves all umbrella-managed Gateway API CRDs in the tenant cluster", func(ctx context.Context) {
			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.List(ctx, &gatewayv1.GatewayClassList{})).To(Succeed())
				g.Expect(vClusterClient.List(ctx, &gatewayv1.GatewayList{}, ctrlclient.InNamespace("default"))).To(Succeed())
				g.Expect(vClusterClient.List(ctx, &gatewayv1.HTTPRouteList{}, ctrlclient.InNamespace("default"))).To(Succeed())
				g.Expect(vClusterClient.List(ctx, &gatewayv1.ReferenceGrantList{}, ctrlclient.InNamespace("default"))).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
		})

		It("syncs tenant Gateways and HTTPRoutes enabled via the umbrella switch", func(ctx context.Context) {
			suffix := random.String(6)
			class := createGatewayClass(ctx, hostClient, "gc-umbrella-"+suffix, umbrellaClassSelectorValue, "umbrella class")
			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Name: class.Name}, &gatewayv1.GatewayClass{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			ns := createTenantNamespace(ctx, vClusterClient, "umbrella-"+suffix)
			var gw *gatewayv1.Gateway
			hostGatewayName := ""

			By("creating a tenant Gateway and expecting it to sync to the host", func() {
				gw = tenantGateway(ns.Name, "gw-umbrella-"+suffix, class.Name)
				Expect(vClusterClient.Create(ctx, gw)).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, gw))).To(Succeed())
				})

				hostGatewayName = translate.SafeConcatName(gw.Name, "x", ns.Name, "x", vClusterName)
				Eventually(func(g Gomega) {
					g.Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostGatewayName}, &gatewayv1.Gateway{})).To(Succeed())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})

			By("creating an HTTPRoute parented to the tenant Gateway and expecting it to sync to the host", func() {
				svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "umbrella-backend-" + suffix, Namespace: ns.Name}, Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}}}
				Expect(vClusterClient.Create(ctx, svc)).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, svc))).To(Succeed())
				})

				route := tenantHTTPRoute(ns.Name, "route-umbrella-"+suffix, gw.Name, svc.Name)
				Expect(vClusterClient.Create(ctx, route)).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, route))).To(Succeed())
				})

				hostRouteName := translate.SafeConcatName(route.Name, "x", ns.Name, "x", vClusterName)
				Eventually(func(g Gomega) {
					got := &gatewayv1.HTTPRoute{}
					g.Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostRouteName}, got)).To(Succeed())
					g.Expect(got.Spec.ParentRefs).To(HaveLen(1))
					g.Expect(got.Spec.ParentRefs[0].Name).To(Equal(gatewayv1.ObjectName(hostGatewayName)))
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})
		})

		It("validates cross-namespace routes against tenant ReferenceGrants without syncing the grants to the host", func(ctx context.Context) {
			suffix := random.String(6)
			class := createGatewayClass(ctx, hostClient, "gc-umbrella-rg-"+suffix, umbrellaClassSelectorValue, "umbrella referencegrant class")
			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Name: class.Name}, &gatewayv1.GatewayClass{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			frontend := createTenantNamespace(ctx, vClusterClient, "umbrella-fe-"+suffix)
			backend := createTenantNamespace(ctx, vClusterClient, "umbrella-be-"+suffix)
			gw := tenantGateway(frontend.Name, "gw-umbrella-rg-"+suffix, class.Name)
			Expect(vClusterClient.Create(ctx, gw)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, gw))).To(Succeed())
			})
			hostGatewayName := translate.SafeConcatName(gw.Name, "x", frontend.Name, "x", vClusterName)
			Eventually(func(g Gomega) {
				g.Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostGatewayName}, &gatewayv1.Gateway{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "umbrella-rg-backend-" + suffix, Namespace: backend.Name}, Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}}}
			Expect(vClusterClient.Create(ctx, svc)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, svc))).To(Succeed())
			})

			var hostRouteName string
			By("creating a route whose backendRef crosses into another namespace and expecting no host sync", func() {
				route := crossNamespaceRoute(frontend.Name, "route-umbrella-rg-"+suffix, gw.Name, backend.Name, svc.Name)
				Expect(vClusterClient.Create(ctx, route)).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, route))).To(Succeed())
				})

				hostRouteName = translate.SafeConcatName(route.Name, "x", frontend.Name, "x", vClusterName)
				Consistently(func(g Gomega) {
					err := hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostRouteName}, &gatewayv1.HTTPRoute{})
					g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
			})

			By("creating a tenant ReferenceGrant and expecting the route to sync while the grant stays virtual", func() {
				grant := &gatewayv1.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{Name: "allow-umbrella-" + suffix, Namespace: backend.Name},
					Spec: gatewayv1.ReferenceGrantSpec{
						From: []gatewayv1.ReferenceGrantFrom{{Group: gatewayv1.GroupName, Kind: "HTTPRoute", Namespace: gatewayv1.Namespace(frontend.Name)}},
						To:   []gatewayv1.ReferenceGrantTo{{Group: "", Kind: "Service"}},
					},
				}
				Expect(vClusterClient.Create(ctx, grant)).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, grant))).To(Succeed())
				})

				Eventually(func(g Gomega) {
					g.Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostRouteName}, &gatewayv1.HTTPRoute{})).To(Succeed())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

				hostGrantName := translate.SafeConcatName(grant.Name, "x", backend.Name, "x", vClusterName)
				Consistently(func(g Gomega) {
					err := hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostGrantName}, &gatewayv1.ReferenceGrant{})
					g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
			})
		})
	})
}
