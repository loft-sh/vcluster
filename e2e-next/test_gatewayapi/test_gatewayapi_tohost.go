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
	"k8s.io/utils/ptr"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayv1alpha3 "sigs.k8s.io/gateway-api/apis/v1alpha3"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// GatewayAPIToHostSpec registers tenant-authored route/policy sync tests
// (P0 cases 13, 18, 19).
func GatewayAPIToHostSpec() {
	Describe("Gateway API toHost", labels.GatewayAPI, func() {
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
			Expect(gatewayv1alpha2.Install(scheme)).To(Succeed())
			Expect(gatewayv1alpha3.Install(scheme)).To(Succeed())
			Expect(gatewayv1beta1.Install(scheme)).To(Succeed())

			hostClient, err = ctrlclient.New(cluster.From(ctx, constants.GetHostClusterName()).KubernetesRestConfig(), ctrlclient.Options{Scheme: scheme})
			Expect(err).To(Succeed())
			vClusterClient, err = ctrlclient.New(cluster.CurrentClusterFrom(ctx).KubernetesRestConfig(), ctrlclient.Options{Scheme: scheme})
			Expect(err).To(Succeed())
			vClusterName = cluster.CurrentClusterNameFrom(ctx)
			vClusterHostNS = "vcluster-" + vClusterName
		})

		It("does not sync cross-namespace backendRef routes until a ReferenceGrant permits it", labels.PR, func(ctx context.Context) {
			suffix := random.String(6)
			class := createGatewayClass(ctx, hostClient, "gc-rg-"+suffix, gatewayClassSelectorValue, "referencegrant class")
			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Name: class.Name}, &gatewayv1.GatewayClass{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			frontend := createTenantNamespace(ctx, vClusterClient, "frontend-"+suffix)
			backend := createTenantNamespace(ctx, vClusterClient, "backend-"+suffix)
			gw := tenantGateway(frontend.Name, "gw-"+suffix, class.Name)
			Expect(vClusterClient.Create(ctx, gw)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, gw))).To(Succeed())
			})
			svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "backend-svc-" + suffix, Namespace: backend.Name}, Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}}}
			Expect(vClusterClient.Create(ctx, svc)).To(Succeed())

			By("creating a route whose backendRef crosses into another namespace")
			route := crossNamespaceRoute(frontend.Name, "route-"+suffix, gw.Name, backend.Name, svc.Name)
			Expect(vClusterClient.Create(ctx, route)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, route))).To(Succeed())
			})

			hostRouteName := translate.SafeConcatName(route.Name, "x", frontend.Name, "x", vClusterName)
			Consistently(func(g Gomega) {
				err := hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostRouteName}, &gatewayv1.HTTPRoute{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryShort).Should(Succeed())

			By("creating a ReferenceGrant in the backend namespace and expecting the route to sync")
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
			Eventually(func(g Gomega) {
				g.Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostRouteName}, &gatewayv1.HTTPRoute{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
		})

		It("syncs opt-in TLSRoutes to the host and propagates updates and deletes", labels.PR, func(ctx context.Context) {
			suffix := random.String(6)
			ns := createTenantNamespace(ctx, vClusterClient, "tls-"+suffix)
			svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "tls-backend-" + suffix, Namespace: ns.Name}, Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 443}}}}
			Expect(vClusterClient.Create(ctx, svc)).To(Succeed())

			route := &gatewayv1alpha2.TLSRoute{
				ObjectMeta: metav1.ObjectMeta{Name: "app-tls-" + suffix, Namespace: ns.Name},
				Spec: gatewayv1alpha2.TLSRouteSpec{
					Hostnames: []gatewayv1.Hostname{"app.apps.example.com"},
					Rules: []gatewayv1alpha2.TLSRouteRule{{BackendRefs: []gatewayv1.BackendRef{{
						BackendObjectReference: gatewayv1.BackendObjectReference{Name: gatewayv1.ObjectName(svc.Name), Port: ptr.To(gatewayv1.PortNumber(443))},
					}}}},
				},
			}
			Expect(vClusterClient.Create(ctx, route)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, route))).To(Succeed())
			})

			hostRouteName := translate.SafeConcatName(route.Name, "x", ns.Name, "x", vClusterName)
			hostKey := types.NamespacedName{Namespace: vClusterHostNS, Name: hostRouteName}
			Eventually(func(g Gomega) {
				got := &gatewayv1alpha2.TLSRoute{}
				g.Expect(hostClient.Get(ctx, hostKey, got)).To(Succeed())
				g.Expect(got.Spec.Hostnames).To(ContainElement(gatewayv1.Hostname("app.apps.example.com")))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			By("patching the tenant TLSRoute hostname and expecting the host update")
			current := &gatewayv1alpha2.TLSRoute{}
			Expect(vClusterClient.Get(ctx, ctrlclient.ObjectKeyFromObject(route), current)).To(Succeed())
			current.Spec.Hostnames = []gatewayv1.Hostname{"app-updated.apps.example.com"}
			Expect(vClusterClient.Update(ctx, current)).To(Succeed())
			Eventually(func(g Gomega) {
				got := &gatewayv1alpha2.TLSRoute{}
				g.Expect(hostClient.Get(ctx, hostKey, got)).To(Succeed())
				g.Expect(got.Spec.Hostnames).To(ContainElement(gatewayv1.Hostname("app-updated.apps.example.com")))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			By("deleting the tenant TLSRoute and expecting host deletion")
			Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, current))).To(Succeed())
			Eventually(func(g Gomega) {
				err := hostClient.Get(ctx, hostKey, &gatewayv1alpha2.TLSRoute{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
		})

		It("syncs opt-in BackendTLSPolicies to the host with a translated targetRef", labels.PR, func(ctx context.Context) {
			suffix := random.String(6)
			ns := createTenantNamespace(ctx, vClusterClient, "btls-"+suffix)
			svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "btls-backend-" + suffix, Namespace: ns.Name}, Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 443}}}}
			Expect(vClusterClient.Create(ctx, svc)).To(Succeed())

			policy := &gatewayv1alpha3.BackendTLSPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "btls-" + suffix, Namespace: ns.Name},
				Spec: gatewayv1.BackendTLSPolicySpec{
					TargetRefs: []gatewayv1.LocalPolicyTargetReferenceWithSectionName{{
						LocalPolicyTargetReference: gatewayv1.LocalPolicyTargetReference{Group: gatewayv1.Group(""), Kind: gatewayv1.Kind("Service"), Name: gatewayv1.ObjectName(svc.Name)},
					}},
					Validation: gatewayv1.BackendTLSPolicyValidation{
						WellKnownCACertificates: ptr.To(gatewayv1.WellKnownCACertificatesSystem),
						Hostname:                gatewayv1.PreciseHostname("backend.apps.example.com"),
					},
				},
			}
			Expect(vClusterClient.Create(ctx, policy)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, policy))).To(Succeed())
			})

			hostPolicyName := translate.SafeConcatName(policy.Name, "x", ns.Name, "x", vClusterName)
			hostKey := types.NamespacedName{Namespace: vClusterHostNS, Name: hostPolicyName}
			Eventually(func(g Gomega) {
				got := &gatewayv1alpha3.BackendTLSPolicy{}
				g.Expect(hostClient.Get(ctx, hostKey, got)).To(Succeed())
				g.Expect(got.Spec.TargetRefs).To(HaveLen(1))
				g.Expect(string(got.Spec.TargetRefs[0].Name)).To(Equal(translate.SafeConcatName(svc.Name, "x", ns.Name, "x", vClusterName)))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			By("updating the policy hostname and expecting the host update")
			current := &gatewayv1alpha3.BackendTLSPolicy{}
			Expect(vClusterClient.Get(ctx, ctrlclient.ObjectKeyFromObject(policy), current)).To(Succeed())
			current.Spec.Validation.Hostname = gatewayv1.PreciseHostname("backend-updated.apps.example.com")
			Expect(vClusterClient.Update(ctx, current)).To(Succeed())
			Eventually(func(g Gomega) {
				got := &gatewayv1alpha3.BackendTLSPolicy{}
				g.Expect(hostClient.Get(ctx, hostKey, got)).To(Succeed())
				g.Expect(got.Spec.Validation.Hostname).To(Equal(gatewayv1.PreciseHostname("backend-updated.apps.example.com")))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			By("deleting the policy and expecting host deletion")
			Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, current))).To(Succeed())
			Eventually(func(g Gomega) {
				err := hostClient.Get(ctx, hostKey, &gatewayv1alpha3.BackendTLSPolicy{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
		})
	})
}

func crossNamespaceRoute(namespace, name, gatewayName, backendNamespace, serviceName string) *gatewayv1.HTTPRoute {
	return &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{ParentRefs: []gatewayv1.ParentReference{{Name: gatewayv1.ObjectName(gatewayName), SectionName: ptr.To(gatewayv1.SectionName("http"))}}},
			Rules: []gatewayv1.HTTPRouteRule{{BackendRefs: []gatewayv1.HTTPBackendRef{{BackendRef: gatewayv1.BackendRef{BackendObjectReference: gatewayv1.BackendObjectReference{
				Name:      gatewayv1.ObjectName(serviceName),
				Namespace: ptr.To(gatewayv1.Namespace(backendNamespace)),
				Port:      ptr.To(gatewayv1.PortNumber(80)),
			}}}}}},
		},
	}
}
