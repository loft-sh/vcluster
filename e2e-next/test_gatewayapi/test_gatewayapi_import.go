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

const importClassSelectorValue = "gatewayapi-import"

// GatewayAPIImportSpec registers fromHost imported Gateway tests (P0 cases
// 01-08, 16, control-plane behavior only).
func GatewayAPIImportSpec() {
	Describe("Gateway API import", labels.GatewayAPI, labels.GatewayClasses, func() {
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

			ensureHostNamespace(ctx, hostClient, "gwapi-host")
			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.List(ctx, &gatewayv1.GatewayClassList{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
		})

		It("imports a host Gateway into the tenant and syncs attached routes to the host Gateway", labels.PR, func(ctx context.Context) {
			suffix := random.String(6)
			class := createGatewayClass(ctx, hostClient, "gwc-import-"+suffix, importClassSelectorValue, "import class")

			By("creating a host Gateway in the wildcard-mapped source namespace")
			hostGW := hostGateway("gwapi-host", "edge-"+suffix, class.Name)
			createHostGateway(ctx, hostClient, hostGW)

			By("verifying the tenant mirror, namespace and GatewayClass exist")
			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Name: class.Name}, &gatewayv1.GatewayClass{})).To(Succeed())
				mirror := &gatewayv1.Gateway{}
				g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Namespace: "gwapi-import", Name: hostGW.Name}, mirror)).To(Succeed())
				g.Expect(mirror.Labels).To(HaveKeyWithValue("vcluster.loft.sh/imported-gateway", "true"))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			By("creating a tenant HTTPRoute attached to the imported Gateway")
			routeNS := createTenantNamespace(ctx, vClusterClient, "gwapi-app-"+suffix)
			svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "backend-" + suffix, Namespace: routeNS.Name}, Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}}}
			Expect(vClusterClient.Create(ctx, svc)).To(Succeed())
			route := importRoute(routeNS.Name, "route-"+suffix, "gwapi-import", hostGW.Name, svc.Name, "app.apps.example.com")
			Expect(vClusterClient.Create(ctx, route)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, route))).To(Succeed())
			})

			By("verifying the host route parentRef points at the host Gateway")
			hostRouteName := translate.SafeConcatName(route.Name, "x", routeNS.Name, "x", vClusterName)
			Eventually(func(g Gomega) {
				got := &gatewayv1.HTTPRoute{}
				g.Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostRouteName}, got)).To(Succeed())
				g.Expect(got.Spec.ParentRefs).To(HaveLen(1))
				g.Expect(got.Spec.ParentRefs[0].Name).To(Equal(gatewayv1.ObjectName(hostGW.Name)))
				g.Expect(got.Spec.ParentRefs[0].Namespace).NotTo(BeNil())
				g.Expect(*got.Spec.ParentRefs[0].Namespace).To(Equal(gatewayv1.Namespace("gwapi-host")))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			By("confirming no tenant-created host Gateway object exists")
			Consistently(func(g Gomega) {
				err := hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: translate.SafeConcatName(hostGW.Name, "x", "gwapi-import", "x", vClusterName)}, &gatewayv1.Gateway{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
		})

		It("mirrors the GatewayClass with controllerName preserved and parametersRef stripped", labels.PR, func(ctx context.Context) {
			suffix := random.String(6)
			class := createGatewayClass(ctx, hostClient, "gwc-sanitize-"+suffix, importClassSelectorValue, "sanitize class")

			Eventually(func(g Gomega) {
				mirror := &gatewayv1.GatewayClass{}
				g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Name: class.Name}, mirror)).To(Succeed())
				g.Expect(mirror.Spec.ControllerName).To(Equal(gatewayControllerName))
				g.Expect(mirror.Spec.ParametersRef).To(BeNil())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
		})

		It("does not sync routes that attach to a tenant-local non-imported Gateway", labels.PR, func(ctx context.Context) {
			suffix := random.String(6)
			class := createGatewayClass(ctx, hostClient, "gwc-local-"+suffix, importClassSelectorValue, "local class")
			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Name: class.Name}, &gatewayv1.GatewayClass{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			routeNS := createTenantNamespace(ctx, vClusterClient, "gwapi-local-"+suffix)
			svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "backend-" + suffix, Namespace: routeNS.Name}, Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}}}
			Expect(vClusterClient.Create(ctx, svc)).To(Succeed())

			By("creating a tenant-local Gateway that is not imported from host")
			localGW := tenantGateway(routeNS.Name, "local-gw-"+suffix, class.Name)
			Expect(vClusterClient.Create(ctx, localGW)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, localGW))).To(Succeed())
			})
			route := importRoute(routeNS.Name, "route-"+suffix, routeNS.Name, localGW.Name, svc.Name, "app.example.com")
			Expect(vClusterClient.Create(ctx, route)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, route))).To(Succeed())
			})

			hostRouteName := translate.SafeConcatName(route.Name, "x", routeNS.Name, "x", vClusterName)
			Consistently(func(g Gomega) {
				err := hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostRouteName}, &gatewayv1.HTTPRoute{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
		})

		It("removes the tenant mirror when the host Gateway is deleted and recovers when recreated", labels.PR, func(ctx context.Context) {
			suffix := random.String(6)
			class := createGatewayClass(ctx, hostClient, "gwc-del-"+suffix, importClassSelectorValue, "deletion class")
			hostGW := hostGateway("gwapi-host", "del-edge-"+suffix, class.Name)
			createHostGateway(ctx, hostClient, hostGW)

			mirrorKey := types.NamespacedName{Namespace: "gwapi-import", Name: hostGW.Name}
			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.Get(ctx, mirrorKey, &gatewayv1.Gateway{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			By("deleting the host Gateway")
			Expect(hostClient.Delete(ctx, hostGW)).To(Succeed())
			Eventually(func(g Gomega) {
				err := vClusterClient.Get(ctx, mirrorKey, &gatewayv1.Gateway{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			By("recreating the host Gateway")
			recreated := hostGateway("gwapi-host", hostGW.Name, class.Name)
			Expect(hostClient.Create(ctx, recreated)).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.Get(ctx, mirrorKey, &gatewayv1.Gateway{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
		})

		It("treats the imported Gateway as read-only and recreates it after tenant deletion", labels.PR, func(ctx context.Context) {
			suffix := random.String(6)
			class := createGatewayClass(ctx, hostClient, "gwc-ro-"+suffix, importClassSelectorValue, "read-only class")
			hostGW := hostGateway("gwapi-host", "ro-edge-"+suffix, class.Name)
			createHostGateway(ctx, hostClient, hostGW)

			mirrorKey := types.NamespacedName{Namespace: "gwapi-import", Name: hostGW.Name}
			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.Get(ctx, mirrorKey, &gatewayv1.Gateway{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			By("editing a tenant listener port and expecting it reverted")
			mirror := &gatewayv1.Gateway{}
			Expect(vClusterClient.Get(ctx, mirrorKey, mirror)).To(Succeed())
			mirror.Spec.Listeners[0].Port = gatewayv1.PortNumber(8080)
			Expect(vClusterClient.Update(ctx, mirror)).To(Succeed())
			Eventually(func(g Gomega) {
				got := &gatewayv1.Gateway{}
				g.Expect(vClusterClient.Get(ctx, mirrorKey, got)).To(Succeed())
				g.Expect(got.Spec.Listeners[0].Port).To(Equal(gatewayv1.PortNumber(80)))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			By("deleting the tenant mirror and expecting recreation")
			toDelete := &gatewayv1.Gateway{}
			Expect(vClusterClient.Get(ctx, mirrorKey, toDelete)).To(Succeed())
			Expect(vClusterClient.Delete(ctx, toDelete)).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.Get(ctx, mirrorKey, &gatewayv1.Gateway{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			By("confirming the host Gateway is never mutated")
			got := &gatewayv1.Gateway{}
			Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: "gwapi-host", Name: hostGW.Name}, got)).To(Succeed())
			Expect(got.Spec.Listeners[0].Port).To(Equal(gatewayv1.PortNumber(80)))
		})

		It("hides host Gateway status addresses on the tenant mirror by default", labels.PR, func(ctx context.Context) {
			suffix := random.String(6)
			class := createGatewayClass(ctx, hostClient, "gwc-status-"+suffix, importClassSelectorValue, "status class")
			hostGW := hostGateway("gwapi-host", "status-edge-"+suffix, class.Name)
			createHostGateway(ctx, hostClient, hostGW)

			mirrorKey := types.NamespacedName{Namespace: "gwapi-import", Name: hostGW.Name}
			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.Get(ctx, mirrorKey, &gatewayv1.Gateway{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			By("populating host Gateway status addresses")
			current := &gatewayv1.Gateway{}
			Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: "gwapi-host", Name: hostGW.Name}, current)).To(Succeed())
			current.Status.Addresses = []gatewayv1.GatewayStatusAddress{{Type: ptr.To(gatewayv1.IPAddressType), Value: "203.0.113.10"}}
			Expect(hostClient.Status().Update(ctx, current)).To(Succeed())

			By("verifying the tenant mirror hides addresses")
			Consistently(func(g Gomega) {
				got := &gatewayv1.Gateway{}
				g.Expect(vClusterClient.Get(ctx, mirrorKey, got)).To(Succeed())
				g.Expect(got.Status.Addresses).To(BeEmpty())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
		})

		It("enforces the virtual allowedRoutes namespace selector policy on imported Gateways", labels.PR, func(ctx context.Context) {
			suffix := random.String(6)
			ensureHostNamespace(ctx, hostClient, "gwapi-host-policy")
			class := createGatewayClass(ctx, hostClient, "gwc-sel-"+suffix, importClassSelectorValue, "selector class")
			hostGW := hostGateway("gwapi-host-policy", "selector-edge", class.Name)
			createHostGateway(ctx, hostClient, hostGW)

			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Namespace: "gwapi-import-policy", Name: "selector-edge"}, &gatewayv1.Gateway{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			By("creating a route in an unlabeled namespace and expecting no host sync")
			routeNS := createTenantNamespace(ctx, vClusterClient, "gwapi-sel-"+suffix)
			svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "backend-" + suffix, Namespace: routeNS.Name}, Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}}}
			Expect(vClusterClient.Create(ctx, svc)).To(Succeed())
			route := importRoute(routeNS.Name, "route-"+suffix, "gwapi-import-policy", "selector-edge", svc.Name, "app.apps.example.com")
			Expect(vClusterClient.Create(ctx, route)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, route))).To(Succeed())
			})
			hostRouteName := translate.SafeConcatName(route.Name, "x", routeNS.Name, "x", vClusterName)
			Consistently(func(g Gomega) {
				err := hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostRouteName}, &gatewayv1.HTTPRoute{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())

			By("labeling the namespace team=apps and expecting the host route to appear")
			updatedNS := &corev1.Namespace{}
			Expect(vClusterClient.Get(ctx, types.NamespacedName{Name: routeNS.Name}, updatedNS)).To(Succeed())
			if updatedNS.Labels == nil {
				updatedNS.Labels = map[string]string{}
			}
			updatedNS.Labels["team"] = "apps"
			Expect(vClusterClient.Update(ctx, updatedNS)).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostRouteName}, &gatewayv1.HTTPRoute{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
		})

		It("enforces the hostname allowlist on imported Gateways", labels.PR, func(ctx context.Context) {
			suffix := random.String(6)
			ensureHostNamespace(ctx, hostClient, "gwapi-host-policy")
			class := createGatewayClass(ctx, hostClient, "gwc-host-"+suffix, importClassSelectorValue, "hostname class")
			hostGW := hostGateway("gwapi-host-policy", "hostname-edge", class.Name)
			createHostGateway(ctx, hostClient, hostGW)

			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Namespace: "gwapi-import-policy", Name: "hostname-edge"}, &gatewayv1.Gateway{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			routeNS := createTenantNamespace(ctx, vClusterClient, "gwapi-host-"+suffix)
			svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "backend-" + suffix, Namespace: routeNS.Name}, Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}}}
			Expect(vClusterClient.Create(ctx, svc)).To(Succeed())

			By("creating a route with a disallowed hostname and expecting no host sync")
			denied := importRoute(routeNS.Name, "denied-"+suffix, "gwapi-import-policy", "hostname-edge", svc.Name, "admin.apps.example.com")
			Expect(vClusterClient.Create(ctx, denied)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, denied))).To(Succeed())
			})
			deniedHostName := translate.SafeConcatName(denied.Name, "x", routeNS.Name, "x", vClusterName)
			Consistently(func(g Gomega) {
				err := hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: deniedHostName}, &gatewayv1.HTTPRoute{})
				g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())

			By("creating a route with an allowed hostname and expecting host sync")
			allowed := importRoute(routeNS.Name, "allowed-"+suffix, "gwapi-import-policy", "hostname-edge", svc.Name, "api.team-a.apps.example.com")
			Expect(vClusterClient.Create(ctx, allowed)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, allowed))).To(Succeed())
			})
			allowedHostName := translate.SafeConcatName(allowed.Name, "x", routeNS.Name, "x", vClusterName)
			Eventually(func(g Gomega) {
				g.Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: allowedHostName}, &gatewayv1.HTTPRoute{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
		})

		It("imports a renamed Gateway and routes attach to the correct host Gateway", labels.PR, func(ctx context.Context) {
			suffix := random.String(6)
			ensureHostNamespace(ctx, hostClient, "gwapi-host-rename")
			class := createGatewayClass(ctx, hostClient, "gwc-rename-"+suffix, importClassSelectorValue, "rename class")
			hostGW := hostGateway("gwapi-host-rename", "source-edge", class.Name)
			createHostGateway(ctx, hostClient, hostGW)

			By("verifying the tenant mirror exists under the renamed name")
			Eventually(func(g Gomega) {
				g.Expect(vClusterClient.Get(ctx, types.NamespacedName{Namespace: "gwapi-import-rename", Name: "renamed-edge"}, &gatewayv1.Gateway{})).To(Succeed())
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			By("attaching a route to the renamed Gateway and checking the host parentRef")
			routeNS := createTenantNamespace(ctx, vClusterClient, "gwapi-rename-"+suffix)
			svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "backend-" + suffix, Namespace: routeNS.Name}, Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}}}
			Expect(vClusterClient.Create(ctx, svc)).To(Succeed())
			route := importRoute(routeNS.Name, "route-"+suffix, "gwapi-import-rename", "renamed-edge", svc.Name, "app.apps.example.com")
			Expect(vClusterClient.Create(ctx, route)).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				Expect(ctrlclient.IgnoreNotFound(vClusterClient.Delete(ctx, route))).To(Succeed())
			})
			hostRouteName := translate.SafeConcatName(route.Name, "x", routeNS.Name, "x", vClusterName)
			Eventually(func(g Gomega) {
				got := &gatewayv1.HTTPRoute{}
				g.Expect(hostClient.Get(ctx, types.NamespacedName{Namespace: vClusterHostNS, Name: hostRouteName}, got)).To(Succeed())
				g.Expect(got.Spec.ParentRefs).To(HaveLen(1))
				g.Expect(got.Spec.ParentRefs[0].Name).To(Equal(gatewayv1.ObjectName("source-edge")))
				g.Expect(got.Spec.ParentRefs[0].Namespace).NotTo(BeNil())
				g.Expect(*got.Spec.ParentRefs[0].Namespace).To(Equal(gatewayv1.Namespace("gwapi-host-rename")))
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
		})
	})
}

func ensureHostNamespace(ctx context.Context, c ctrlclient.Client, name string) {
	GinkgoHelper()
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	err := c.Create(ctx, ns)
	if err != nil && !kerrors.IsAlreadyExists(err) {
		Expect(err).To(Succeed())
	}
}

func createHostGateway(ctx context.Context, c ctrlclient.Client, gw *gatewayv1.Gateway) {
	GinkgoHelper()
	key := ctrlclient.ObjectKeyFromObject(gw)
	stale := &gatewayv1.Gateway{}
	err := c.Get(ctx, key, stale)
	if err == nil {
		Expect(c.Delete(ctx, stale)).To(Succeed())
		Eventually(func(g Gomega) {
			err := c.Get(ctx, key, &gatewayv1.Gateway{})
			g.Expect(kerrors.IsNotFound(err)).To(BeTrue())
		}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
	} else {
		Expect(kerrors.IsNotFound(err)).To(BeTrue())
	}
	Expect(c.Create(ctx, gw)).To(Succeed())
	DeferCleanup(func(ctx context.Context) { Expect(ctrlclient.IgnoreNotFound(c.Delete(ctx, gw))).To(Succeed()) })
}

func hostGateway(namespace, name, className string) *gatewayv1.Gateway {
	return &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: gatewayv1.ObjectName(className),
			Listeners: []gatewayv1.Listener{{
				Name:     gatewayv1.SectionName("http"),
				Protocol: gatewayv1.HTTPProtocolType,
				Port:     gatewayv1.PortNumber(80),
			}},
		},
	}
}

func importRoute(namespace, name, gatewayNamespace, gatewayName, serviceName, hostname string) *gatewayv1.HTTPRoute {
	return &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{ParentRefs: []gatewayv1.ParentReference{{
				Name:      gatewayv1.ObjectName(gatewayName),
				Namespace: ptr.To(gatewayv1.Namespace(gatewayNamespace)),
			}}},
			Hostnames: []gatewayv1.Hostname{gatewayv1.Hostname(hostname)},
			Rules:     []gatewayv1.HTTPRouteRule{{BackendRefs: []gatewayv1.HTTPBackendRef{{BackendRef: gatewayv1.BackendRef{BackendObjectReference: gatewayv1.BackendObjectReference{Name: gatewayv1.ObjectName(serviceName), Port: ptr.To(gatewayv1.PortNumber(80))}}}}}},
		},
	}
}
