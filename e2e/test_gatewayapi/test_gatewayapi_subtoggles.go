package test_gatewayapi

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
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

const (
	selectiveGatewayClassSelectorValue  = "gatewayapi-selective"
	rgDisabledGatewayClassSelectorValue = "gatewayapi-rgdisabled"
)

func GatewayAPISelectiveSpec() {
	Describe("Gateway API selective toHost", labels.GatewayAPI, func() {
		var clients gatewayAPIClients

		BeforeEach(func(ctx context.Context) {
			installTenantGatewayAPICRDs(ctx, cluster.CurrentClusterFrom(ctx).GetKubeconfig(), tenantHTTPRouteCRD, tenantReferenceGrantCRD)
			clients = newGatewayAPIClients(ctx)
		})

		It("syncs only Gateway when sub-toggles disable httpRoutes and referenceGrants", func(ctx context.Context) {
			suffix := random.String(6)
			class := createGatewayClass(ctx, clients.HostClient, "gc-selective-"+suffix, selectiveGatewayClassSelectorValue, "selective class")
			Eventually(func(g Gomega) {
				g.Expect(clients.VClusterClient.Get(ctx, types.NamespacedName{Name: class.Name}, &gatewayv1.GatewayClass{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			frontend := createTenantNamespace(ctx, clients.VClusterClient, "selective-front-"+suffix)
			backend := createTenantNamespace(ctx, clients.VClusterClient, "selective-back-"+suffix)

			gw := tenantGateway(frontend.Name, "gw-"+suffix, class.Name)
			Expect(clients.VClusterClient.Create(ctx, gw)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(clients.VClusterClient.Delete(ctx, gw))).To(Succeed())
			})
			svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "backend-" + suffix, Namespace: backend.Name}, Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}}}
			Expect(clients.VClusterClient.Create(ctx, svc)).To(Succeed())

			route := crossNamespaceRoute(frontend.Name, "route-"+suffix, gw.Name, backend.Name, svc.Name)
			Expect(clients.VClusterClient.Create(ctx, route)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(clients.VClusterClient.Delete(ctx, route))).To(Succeed())
			})

			grant := &gatewayv1.ReferenceGrant{
				ObjectMeta: metav1.ObjectMeta{Name: "allow-" + suffix, Namespace: backend.Name},
				Spec: gatewayv1.ReferenceGrantSpec{
					From: []gatewayv1.ReferenceGrantFrom{{Group: gatewayv1.Group(gatewayv1.GroupName), Kind: gatewayv1.Kind("HTTPRoute"), Namespace: gatewayv1.Namespace(frontend.Name)}},
					To:   []gatewayv1.ReferenceGrantTo{{Group: gatewayv1.Group(""), Kind: gatewayv1.Kind("Service")}},
				},
			}
			Expect(clients.VClusterClient.Create(ctx, grant)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(clients.VClusterClient.Delete(ctx, grant))).To(Succeed())
			})

			hostGWName := translate.SafeConcatName(gw.Name, "x", frontend.Name, "x", clients.VClusterName)
			hostRouteName := translate.SafeConcatName(route.Name, "x", frontend.Name, "x", clients.VClusterName)
			hostGrantName := translate.SafeConcatName(grant.Name, "x", backend.Name, "x", clients.VClusterName)

			By("verifying the host receives the Gateway", func() {
				Eventually(func(g Gomega) {
					g.Expect(clients.HostClient.Get(ctx, types.NamespacedName{Namespace: clients.VClusterHostNS, Name: hostGWName}, &gatewayv1.Gateway{})).To(Succeed())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})

			By("verifying the host does not receive the HTTPRoute or ReferenceGrant", func() {
				Consistently(func(g Gomega) {
					err := clients.HostClient.Get(ctx, types.NamespacedName{Namespace: clients.VClusterHostNS, Name: hostRouteName}, &gatewayv1.HTTPRoute{})
					g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "HTTPRoute should not be on host when httpRoutes.enabled=false")
					err = clients.HostClient.Get(ctx, types.NamespacedName{Namespace: clients.VClusterHostNS, Name: hostGrantName}, &gatewayv1.ReferenceGrant{})
					g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "ReferenceGrant should not be on host when referenceGrants.enabled=false")
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
			})
		})
	})
}

func GatewayAPIReferenceGrantDisabledSpec() {
	Describe("Gateway API referenceGrants disabled", labels.GatewayAPI, func() {
		var clients gatewayAPIClients

		BeforeEach(func(ctx context.Context) {
			installTenantGatewayAPICRDs(ctx, cluster.CurrentClusterFrom(ctx).GetKubeconfig(), tenantReferenceGrantCRD)
			clients = newGatewayAPIClients(ctx)
		})

		It("does not sync tenant-created ReferenceGrants when referenceGrants.enabled is false", func(ctx context.Context) {
			suffix := random.String(6)
			class := createGatewayClass(ctx, clients.HostClient, "gc-rgdis-"+suffix, rgDisabledGatewayClassSelectorValue, "rg-disabled class")
			Eventually(func(g Gomega) {
				g.Expect(clients.VClusterClient.Get(ctx, types.NamespacedName{Name: class.Name}, &gatewayv1.GatewayClass{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			backend := createTenantNamespace(ctx, clients.VClusterClient, "rgdis-backend-"+suffix)
			grant := &gatewayv1.ReferenceGrant{
				ObjectMeta: metav1.ObjectMeta{Name: "allow-" + suffix, Namespace: backend.Name},
				Spec: gatewayv1.ReferenceGrantSpec{
					From: []gatewayv1.ReferenceGrantFrom{{Group: gatewayv1.Group(gatewayv1.GroupName), Kind: gatewayv1.Kind("HTTPRoute"), Namespace: gatewayv1.Namespace("rgdis-frontend-" + suffix)}},
					To:   []gatewayv1.ReferenceGrantTo{{Group: gatewayv1.Group(""), Kind: gatewayv1.Kind("Service")}},
				},
			}
			Expect(clients.VClusterClient.Create(ctx, grant)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(clients.VClusterClient.Delete(ctx, grant))).To(Succeed())
			})

			hostGrantName := translate.SafeConcatName(grant.Name, "x", backend.Name, "x", clients.VClusterName)
			Consistently(func(g Gomega) {
				err := clients.HostClient.Get(ctx, types.NamespacedName{Namespace: clients.VClusterHostNS, Name: hostGrantName}, &gatewayv1.ReferenceGrant{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "tenant ReferenceGrant must not sync when referenceGrants.enabled is false")
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())

			By("confirming the tenant still owns the ReferenceGrant", func() {
				got := &gatewayv1.ReferenceGrant{}
				Expect(clients.VClusterClient.Get(ctx, ctrlclient.ObjectKeyFromObject(grant), got)).To(Succeed())
				Expect(got.Spec.From).To(HaveLen(1))
			})
		})
	})
}
