package test_gatewayapi

import (
	"context"

	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
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
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

const grantsDisabledGatewayClassSelectorValue = "gatewayapi-grants-disabled-vcluster"

// GatewayAPIGrantsDisabledSpec registers route sync tests for a vCluster with
// sync.toHost.gatewayApi.referenceGrants.enabled set to "false" while route
// sync stays on. Disabling grant sync must not break the route controllers and
// must keep virtual ReferenceGrants authoritative for cross-namespace refs.
func GatewayAPIGrantsDisabledSpec() {
	Describe("Gateway API toHost with ReferenceGrant sync disabled", labels.GatewayAPI, func() {
		var (
			hostClient     ctrlclient.Client
			vClusterClient ctrlclient.Client
			vClusterName   string
			vClusterHostNS string
		)

		BeforeEach(func(ctx context.Context) {
			clients := newGatewayAPIClients(ctx, true)
			hostClient = clients.HostClient
			vClusterClient = clients.VClusterClient
			vClusterName = clients.VClusterName
			vClusterHostNS = clients.VClusterHostNS

			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.List(ctx, &gatewayv1.HTTPRouteList{}, ctrlclient.InNamespace("default"))).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
		})

		It("syncs HTTPRoutes to the host while ReferenceGrant sync is disabled", func(ctx context.Context) {
			suffix := random.String(6)
			class := createGatewayClass(ctx, hostClient, "gc-rgd-"+suffix, grantsDisabledGatewayClassSelectorValue, "grants disabled class")
			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Name: class.Name}, &gatewayv1.GatewayClass{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			ns := createTenantNamespace(ctx, vClusterClient, "rgd-app-"+suffix)
			gw := tenantGateway(ns.Name, "gw-rgd-"+suffix, class.Name)
			Expect(vClusterClient.Create(ctx, gw)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, gw))).To(Succeed())
			})
			hostGatewayName := translate.SafeConcatName(gw.Name, "x", ns.Name, "x", vClusterName)
			Eventually(func(g Gomega) {
				g.Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostGatewayName}, &gatewayv1.Gateway{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "rgd-backend-" + suffix, Namespace: ns.Name}, Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}}}
			Expect(vClusterClient.Create(ctx, svc)).To(Succeed())

			route := tenantHTTPRoute(ns.Name, "route-rgd-"+suffix, gw.Name, svc.Name)
			Expect(vClusterClient.Create(ctx, route)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, route))).To(Succeed())
			})

			hostRouteName := translate.SafeConcatName(route.Name, "x", ns.Name, "x", vClusterName)
			Eventually(func(g Gomega) {
				g.Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostRouteName}, &gatewayv1.HTTPRoute{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
		})

		It("authorizes cross-namespace backendRefs via virtual ReferenceGrants without syncing grants to the host", func(ctx context.Context) {
			suffix := random.String(6)
			class := createGatewayClass(ctx, hostClient, "gc-rgd-xns-"+suffix, grantsDisabledGatewayClassSelectorValue, "grants disabled cross-namespace class")
			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Name: class.Name}, &gatewayv1.GatewayClass{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			frontend := createTenantNamespace(ctx, vClusterClient, "rgd-frontend-"+suffix)
			backend := createTenantNamespace(ctx, vClusterClient, "rgd-backend-"+suffix)
			gw := tenantGateway(frontend.Name, "gw-rgd-xns-"+suffix, class.Name)
			Expect(vClusterClient.Create(ctx, gw)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, gw))).To(Succeed())
			})
			hostGatewayName := translate.SafeConcatName(gw.Name, "x", frontend.Name, "x", vClusterName)
			Eventually(func(g Gomega) {
				g.Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostGatewayName}, &gatewayv1.Gateway{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "rgd-xns-backend-" + suffix, Namespace: backend.Name}, Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}}}
			Expect(vClusterClient.Create(ctx, svc)).To(Succeed())

			var hostRouteName string
			var route *gatewayv1.HTTPRoute
			var grant *gatewayv1beta1.ReferenceGrant

			By("creating a route whose backendRef crosses into another namespace and expecting no host sync", func() {
				route = crossNamespaceRoute(frontend.Name, "route-rgd-xns-"+suffix, gw.Name, backend.Name, svc.Name)
				Expect(vClusterClient.Create(ctx, route)).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, route))).To(Succeed())
				})

				hostRouteName = translate.SafeConcatName(route.Name, "x", frontend.Name, "x", vClusterName)
				Consistently(func(g Gomega) {
					err := hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostRouteName}, &gatewayv1.HTTPRoute{})
					g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "host route %s should stay absent until a grant permits it, got error: %v", hostRouteName, err)
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
			})

			By("creating a virtual ReferenceGrant and expecting the route to sync", func() {
				// The ReferenceGrant CRD must be served in the tenant cluster even
				// with grant sync disabled — virtual grants still govern
				// cross-namespace authorization.
				Eventually(func(g Gomega) {
					g.Expect(vClusterClient.List(ctx, &gatewayv1beta1.ReferenceGrantList{}, ctrlclient.InNamespace(backend.Name))).To(Succeed())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

				grant = &gatewayv1beta1.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{Name: "allow-rgd-" + suffix, Namespace: backend.Name},
					Spec: gatewayv1beta1.ReferenceGrantSpec{
						From: []gatewayv1beta1.ReferenceGrantFrom{{Group: gatewayv1.GroupName, Kind: "HTTPRoute", Namespace: gatewayv1.Namespace(frontend.Name)}},
						To:   []gatewayv1beta1.ReferenceGrantTo{{Group: "", Kind: "Service"}},
					},
				}
				Expect(vClusterClient.Create(ctx, grant)).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, grant))).To(Succeed())
				})
				Eventually(func(g Gomega) {
					g.Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostRouteName}, &gatewayv1.HTTPRoute{})).To(Succeed())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})

			By("expecting the virtual ReferenceGrant to never sync to the host", func() {
				hostGrantName := translate.SafeConcatName(grant.Name, "x", backend.Name, "x", vClusterName)
				Consistently(func(g Gomega) {
					err := hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostGrantName}, &gatewayv1beta1.ReferenceGrant{})
					g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "host grant %s should not exist with grant sync disabled, got error: %v", hostGrantName, err)
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
			})
		})
	})
}
