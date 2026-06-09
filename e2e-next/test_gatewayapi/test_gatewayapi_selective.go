package test_gatewayapi

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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

const selectiveGatewayClassSelectorValue = "gatewayapi-selective"

// GatewayAPISelectiveSpec registers tests for the broken-out
// toHost.gatewayApi sub-toggle form (TC-02a variant B). Only gateways are
// enabled here; HTTPRoutes and ReferenceGrants must NOT sync to the host.
func GatewayAPISelectiveSpec() {
	Describe("Gateway API selective toHost", labels.GatewayAPI, func() {
		var (
			hostClient     ctrlclient.Client
			vClusterClient ctrlclient.Client
			vClusterName   string
			vClusterHostNS string
		)

		BeforeEach(func(ctx context.Context) {
			var err error
			scheme := runtime.NewScheme()
			Expect(corev1.AddToScheme(scheme)).To(Succeed())
			Expect(gatewayv1.Install(scheme)).To(Succeed())
			Expect(gatewayv1beta1.Install(scheme)).To(Succeed())

			hostClient, err = ctrlclient.New(cluster.From(ctx, constants.GetHostClusterName()).KubernetesRestConfig(), ctrlclient.Options{Scheme: scheme})
			Expect(err).To(Succeed())
			vClusterClient, err = ctrlclient.New(cluster.CurrentClusterFrom(ctx).KubernetesRestConfig(), ctrlclient.Options{Scheme: scheme})
			Expect(err).To(Succeed())
			vClusterName = cluster.CurrentClusterNameFrom(ctx)
			vClusterHostNS = "vcluster-" + vClusterName

			// vCluster does not install HTTPRoute/ReferenceGrant CRDs in the tenant
			// because their sync sub-toggles are off — install them so the tenant
			// can attempt to create the resources we're verifying don't sync.
			installTenantGatewayAPICRDs(ctx, cluster.CurrentClusterFrom(ctx).GetKubeconfig(), tenantHTTPRouteCRD, tenantReferenceGrantCRD)
		})

		It("syncs only Gateway when sub-toggles disable httpRoutes and referenceGrants", labels.PR, func(ctx context.Context) {
			suffix := random.String(6)
			class := createGatewayClass(ctx, hostClient, "gc-selective-"+suffix, selectiveGatewayClassSelectorValue, "selective class")
			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Name: class.Name}, &gatewayv1.GatewayClass{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			frontend := createTenantNamespace(ctx, vClusterClient, "selective-front-"+suffix)
			backend := createTenantNamespace(ctx, vClusterClient, "selective-back-"+suffix)

			gw := tenantGateway(frontend.Name, "gw-"+suffix, class.Name)
			Expect(vClusterClient.Create(ctx, gw)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, gw))).To(Succeed())
			})
			svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "backend-" + suffix, Namespace: backend.Name}, Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}}}
			Expect(vClusterClient.Create(ctx, svc)).To(Succeed())

			route := crossNamespaceRoute(frontend.Name, "route-"+suffix, gw.Name, backend.Name, svc.Name)
			Expect(vClusterClient.Create(ctx, route)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, route))).To(Succeed())
			})

			grant := &gatewayv1beta1.ReferenceGrant{
				ObjectMeta: metav1.ObjectMeta{Name: "allow-" + suffix, Namespace: backend.Name},
				Spec: gatewayv1beta1.ReferenceGrantSpec{
					From: []gatewayv1beta1.ReferenceGrantFrom{{Group: gatewayv1.Group(gatewayv1.GroupName), Kind: gatewayv1.Kind("HTTPRoute"), Namespace: gatewayv1.Namespace(frontend.Name)}},
					To:   []gatewayv1beta1.ReferenceGrantTo{{Group: gatewayv1.Group(""), Kind: gatewayv1.Kind("Service")}},
				},
			}
			Expect(vClusterClient.Create(ctx, grant)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, grant))).To(Succeed())
			})

			hostGWName := translate.SafeConcatName(gw.Name, "x", frontend.Name, "x", vClusterName)
			hostRouteName := translate.SafeConcatName(route.Name, "x", frontend.Name, "x", vClusterName)
			hostGrantName := translate.SafeConcatName(grant.Name, "x", backend.Name, "x", vClusterName)

			By("verifying the host receives the Gateway", func() {
				Eventually(func(g Gomega) {
					g.Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostGWName}, &gatewayv1.Gateway{})).To(Succeed())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})

			By("verifying the host does not receive the HTTPRoute or ReferenceGrant", func() {
				Consistently(func(g Gomega) {
					err := hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostRouteName}, &gatewayv1.HTTPRoute{})
					g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "HTTPRoute should not be on host when httpRoutes.enabled=false")
					err = hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostGrantName}, &gatewayv1beta1.ReferenceGrant{})
					g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "ReferenceGrant should not be on host when referenceGrants.enabled=false")
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
			})
		})
	})
}
