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
)

const (
	gatewayClassSelectorKey   = "e2e.vcluster.loft.sh/gatewayclass"
	gatewayClassSelectorValue = "gatewayapi-vcluster"
	gatewayControllerName     = gatewayv1.GatewayController("example.com/gateway-controller")
)

// GatewayAPISyncSpec registers Gateway API sync tests.
func GatewayAPISyncSpec() {
	Describe("Gateway API sync", labels.GatewayAPI, labels.GatewayClasses, func() {
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

			hostClient, err = ctrlclient.New(cluster.From(ctx, constants.GetHostClusterName()).KubernetesRestConfig(), ctrlclient.Options{Scheme: scheme})
			Expect(err).To(Succeed())
			vClusterClient, err = ctrlclient.New(cluster.CurrentClusterFrom(ctx).KubernetesRestConfig(), ctrlclient.Options{Scheme: scheme})
			Expect(err).To(Succeed())
			vClusterName = cluster.CurrentClusterNameFrom(ctx)
			vClusterHostNS = "vcluster-" + vClusterName

			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.List(ctx, &gatewayv1.GatewayClassList{})).To(Succeed())
				g.Expect(vClusterClient.List(ctx, &gatewayv1.GatewayList{}, ctrlclient.InNamespace("default"))).To(Succeed())
				g.Expect(vClusterClient.List(ctx, &gatewayv1.HTTPRouteList{}, ctrlclient.InNamespace("default"))).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
		})

		It("mirrors matching Host GatewayClasses into the Tenant Cluster and hides non-matching ones", func(ctx context.Context) {
			suffix := random.String(6)
			allowed, hidden := createGatewayClassPair(ctx, hostClient, suffix)

			By("waiting for the allowed GatewayClass to appear sanitized in the Tenant Cluster", func() {
				Eventually(func(g Gomega) {
					got := &gatewayv1.GatewayClass{}
					g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Name: allowed.Name}, got)).To(Succeed())
					g.Expect(got.Spec.ControllerName).To(Equal(gatewayControllerName))
					g.Expect(got.Spec.Description).NotTo(BeNil())
					g.Expect(*got.Spec.Description).To(Equal("e2e allowed GatewayClass"))
					g.Expect(got.Spec.ParametersRef).To(BeNil())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})

			By("proving the selector-hidden GatewayClass stays absent from the Tenant Cluster", func() {
				Consistently(func(g Gomega) {
					err := vClusterClient.Get(ctx, types.NamespacedName{Name: hidden.Name}, &gatewayv1.GatewayClass{})
					g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
			})

			By("updating the Host GatewayClass and ensuring parametersRef remains sanitized", func() {
				Expect(hostClient.Get(ctx, types.NamespacedName{Name: allowed.Name}, allowed)).To(Succeed())
				if allowed.Annotations == nil {
					allowed.Annotations = map[string]string{}
				}
				allowed.Annotations["e2e.vcluster.loft.sh/resync"] = suffix
				Expect(hostClient.Update(ctx, allowed)).To(Succeed())
				Eventually(func(g Gomega) {
					got := &gatewayv1.GatewayClass{}
					g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Name: allowed.Name}, got)).To(Succeed())
					g.Expect(got.Annotations).To(HaveKeyWithValue("e2e.vcluster.loft.sh/resync", suffix))
					g.Expect(got.Spec.ParametersRef).To(BeNil())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})

			By("deleting the Host GatewayClass and waiting for the Tenant mirror to disappear", func() {
				Expect(ctrlclient.IgnoreNotFound(hostClient.Delete(ctx, allowed))).To(Succeed())
				Eventually(func(g Gomega) {
					err := vClusterClient.Get(ctx, types.NamespacedName{Name: allowed.Name}, &gatewayv1.GatewayClass{})
					g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})
		})

		It("does not sync Tenant Gateways that reference unavailable GatewayClasses", func(ctx context.Context) {
			suffix := random.String(6)
			allowed, hidden := createGatewayClassPair(ctx, hostClient, suffix)
			ns := createTenantNamespace(ctx, vClusterClient, "gwapi-reject-"+suffix)

			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Name: allowed.Name}, &gatewayv1.GatewayClass{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			bad := tenantGateway(ns.Name, "bad-gw-"+suffix, hidden.Name)
			Expect(vClusterClient.Create(ctx, bad)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, bad))).To(Succeed())
			})
			badHostName := translate.SafeConcatName(bad.Name, "x", ns.Name, "x", vClusterName)
			Consistently(func(g Gomega) {
				err := hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: badHostName}, &gatewayv1.Gateway{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())

			good := tenantGateway(ns.Name, "good-gw-"+suffix, allowed.Name)
			Expect(vClusterClient.Create(ctx, good)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, good))).To(Succeed())
			})
			goodHostName := translate.SafeConcatName(good.Name, "x", ns.Name, "x", vClusterName)
			Eventually(func(g Gomega) {
				got := &gatewayv1.Gateway{}
				g.Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: goodHostName}, got)).To(Succeed())
				g.Expect(got.Spec.GatewayClassName).To(Equal(gatewayv1.ObjectName(allowed.Name)))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			Expect(vClusterClient.Get(ctx, ctrlclient.ObjectKeyFromObject(good), good)).To(Succeed())
			good.Spec.GatewayClassName = gatewayv1.ObjectName(hidden.Name)
			Expect(vClusterClient.Update(ctx, good)).To(Succeed())
			Eventually(func(g Gomega) {
				err := hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: goodHostName}, &gatewayv1.Gateway{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
		})

		It("syncs Tenant Gateway and HTTPRoute to Host for an available GatewayClass", func(ctx context.Context) {
			suffix := random.String(6)
			allowed := createGatewayClass(ctx, hostClient, "gc-allowed-"+suffix, gatewayClassSelectorValue, "e2e allowed GatewayClass")
			ns := createTenantNamespace(ctx, vClusterClient, "gwapi-sync-"+suffix)
			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Name: allowed.Name}, &gatewayv1.GatewayClass{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "backend-" + suffix, Namespace: ns.Name}, Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}}}
			Expect(vClusterClient.Create(ctx, service)).To(Succeed())
			gateway := tenantGateway(ns.Name, "gw-"+suffix, allowed.Name)
			Expect(vClusterClient.Create(ctx, gateway)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, gateway))).To(Succeed())
			})

			hostGatewayName := translate.SafeConcatName(gateway.Name, "x", ns.Name, "x", vClusterName)
			Eventually(func(g Gomega) {
				got := &gatewayv1.Gateway{}
				g.Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostGatewayName}, got)).To(Succeed())
				g.Expect(got.Spec.GatewayClassName).To(Equal(gatewayv1.ObjectName(allowed.Name)))
				g.Expect(got.Spec.Listeners).To(HaveLen(1))
				listener := got.Spec.Listeners[0]
				g.Expect(listener.Name).To(Equal(gatewayv1.SectionName("http")))
				g.Expect(listener.Protocol).To(Equal(gatewayv1.HTTPProtocolType))
				g.Expect(listener.Port).To(Equal(gatewayv1.PortNumber(80)))
				g.Expect(listener.Hostname).NotTo(BeNil())
				g.Expect(*listener.Hostname).To(Equal(gatewayv1.Hostname("app.example.com")))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			route := tenantHTTPRoute(ns.Name, "route-"+suffix, gateway.Name, service.Name)
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
				g.Expect(got.Spec.ParentRefs[0].SectionName).NotTo(BeNil())
				g.Expect(*got.Spec.ParentRefs[0].SectionName).To(Equal(gatewayv1.SectionName("http")))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, route))).To(Succeed())
			Eventually(func(g Gomega) {
				err := hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostRouteName}, &gatewayv1.HTTPRoute{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, gateway))).To(Succeed())
			Eventually(func(g Gomega) {
				err := hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostGatewayName}, &gatewayv1.Gateway{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
		})
	})
}

func createGatewayClassPair(ctx context.Context, hostClient ctrlclient.Client, suffix string) (*gatewayv1.GatewayClass, *gatewayv1.GatewayClass) {
	GinkgoHelper()
	allowed := createGatewayClass(ctx, hostClient, "gc-allowed-"+suffix, gatewayClassSelectorValue, "e2e allowed GatewayClass")
	hidden := createGatewayClass(ctx, hostClient, "gc-hidden-"+suffix, "other-"+suffix, "e2e hidden GatewayClass")
	return allowed, hidden
}

func createGatewayClass(ctx context.Context, hostClient ctrlclient.Client, name, selectorValue, description string) *gatewayv1.GatewayClass {
	GinkgoHelper()
	gc := &gatewayv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{gatewayClassSelectorKey: selectorValue}},
		Spec: gatewayv1.GatewayClassSpec{
			ControllerName: gatewayControllerName,
			Description:    ptr.To(description),
			ParametersRef:  gatewayClassParametersRef(name),
		},
	}
	Expect(hostClient.Create(ctx, gc)).To(Succeed())
	DeferCleanup(func(ctx context.Context) { Expect(ctrlclient.IgnoreNotFound(hostClient.Delete(ctx, gc))).To(Succeed()) })
	return gc
}

func gatewayClassParametersRef(name string) *gatewayv1.ParametersReference {
	ns := gatewayv1.Namespace("host-only-" + name)
	return &gatewayv1.ParametersReference{Group: gatewayv1.Group("example.com"), Kind: gatewayv1.Kind("GatewayClassConfig"), Name: name + "-config", Namespace: &ns}
}

func createTenantNamespace(ctx context.Context, c ctrlclient.Client, name string) *corev1.Namespace {
	GinkgoHelper()
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	Expect(c.Create(ctx, ns)).To(Succeed())
	DeferCleanup(func(ctx context.Context) { Expect(ctrlclient.IgnoreNotFound(c.Delete(ctx, ns))).To(Succeed()) })
	return ns
}

func tenantGateway(namespace, name, className string) *gatewayv1.Gateway {
	return &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: gatewayv1.ObjectName(className),
			Listeners: []gatewayv1.Listener{{
				Name:     gatewayv1.SectionName("http"),
				Protocol: gatewayv1.HTTPProtocolType,
				Port:     gatewayv1.PortNumber(80),
				Hostname: ptr.To(gatewayv1.Hostname("app.example.com")),
			}},
		},
	}
}

func tenantHTTPRoute(namespace, name, gatewayName, serviceName string) *gatewayv1.HTTPRoute {
	return &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{ParentRefs: []gatewayv1.ParentReference{{Name: gatewayv1.ObjectName(gatewayName), SectionName: ptr.To(gatewayv1.SectionName("http"))}}},
			Rules:           []gatewayv1.HTTPRouteRule{{BackendRefs: []gatewayv1.HTTPBackendRef{{BackendRef: gatewayv1.BackendRef{BackendObjectReference: gatewayv1.BackendObjectReference{Name: gatewayv1.ObjectName(serviceName), Port: ptr.To(gatewayv1.PortNumber(80))}}}}}},
		},
	}
}
