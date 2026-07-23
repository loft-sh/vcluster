package httproutes

import (
	"strings"
	"testing"

	pkgconfig "github.com/loft-sh/vcluster/pkg/config"
	routetranslate "github.com/loft-sh/vcluster/pkg/controllers/resources/gatewayroutes/translate"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	utiltranslate "github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestSpecToHostTranslatesRuleFilterExtensionRef(t *testing.T) {
	syncCtx := newHTTPRouteTranslateSyncContext(t,
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "filter-config"}},
	)
	expected := utiltranslate.Default.HostName(syncCtx, "filter-config", "team-a")
	route := httpRouteWithRuleFilterExtensionRef(gatewayv1.LocalObjectReference{Group: corev1.GroupName, Kind: "ConfigMap", Name: "filter-config"})

	spec, err := specToHost(syncCtx, route, true)
	if err != nil {
		t.Fatalf("translate HTTPRoute spec: %v", err)
	}
	got := spec.Rules[0].Filters[0].ExtensionRef
	if got == nil {
		t.Fatalf("expected rule filter extensionRef")
	}
	if got.Name != gatewayv1.ObjectName(expected.Name) {
		t.Fatalf("expected rule filter extensionRef name %q, got %q", expected.Name, got.Name)
	}
	if route.Spec.Rules[0].Filters[0].ExtensionRef.Name != "filter-config" {
		t.Fatalf("expected virtual HTTPRoute to stay unchanged, got %q", route.Spec.Rules[0].Filters[0].ExtensionRef.Name)
	}
}

func TestSpecToHostTranslatesBackendRefFilterExtensionRef(t *testing.T) {
	syncCtx := newHTTPRouteTranslateSyncContext(t,
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "backend"}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "filter-secret"}},
	)
	expected := utiltranslate.Default.HostName(syncCtx, "filter-secret", "team-a")
	route := httpRouteWithBackendRefFilterExtensionRef(gatewayv1.LocalObjectReference{Group: corev1.GroupName, Kind: "Secret", Name: "filter-secret"})

	spec, err := specToHost(syncCtx, route, true)
	if err != nil {
		t.Fatalf("translate HTTPRoute spec: %v", err)
	}
	got := spec.Rules[0].BackendRefs[0].Filters[0].ExtensionRef
	if got == nil {
		t.Fatalf("expected backendRef filter extensionRef")
	}
	if got.Name != gatewayv1.ObjectName(expected.Name) {
		t.Fatalf("expected backendRef filter extensionRef name %q, got %q", expected.Name, got.Name)
	}
	if route.Spec.Rules[0].BackendRefs[0].Filters[0].ExtensionRef.Name != "filter-secret" {
		t.Fatalf("expected virtual HTTPRoute to stay unchanged, got %q", route.Spec.Rules[0].BackendRefs[0].Filters[0].ExtensionRef.Name)
	}
}

func TestSpecToHostRequiresManagedHostFilterExtensionRef(t *testing.T) {
	syncCtx := newHTTPRouteTranslateSyncContext(t)
	route := httpRouteWithRuleFilterExtensionRef(gatewayv1.LocalObjectReference{Group: corev1.GroupName, Kind: "ConfigMap", Name: "filter-config"})

	_, err := specToHost(syncCtx, route, true)
	if err == nil || !strings.Contains(err.Error(), "has no synced host object") || !strings.Contains(err.Error(), "extensionRef") {
		t.Fatalf("expected missing host ConfigMap to reject extensionRef with field context, got %v", err)
	}
}

func TestSpecToHostRejectsUnsupportedFilterExtensionRef(t *testing.T) {
	syncCtx := newHTTPRouteTranslateSyncContext(t)
	route := httpRouteWithRuleFilterExtensionRef(gatewayv1.LocalObjectReference{Group: gatewayv1.Group(gatewayv1.GroupVersion.Group), Kind: "GatewayClass", Name: "filter-config"})

	_, err := specToHost(syncCtx, route, true)
	if !routetranslate.IsUnsupportedReference(err) {
		t.Fatalf("expected unsupported extensionRef to be terminal, got %v", err)
	}
}

func newHTTPRouteTranslateSyncContext(t *testing.T, virtualHostObjects ...runtime.Object) *synccontext.SyncContext {
	t.Helper()
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	seedCtx := syncertesting.NewFakeRegisterContext(vcConfig, testingutil.NewFakeClient(scheme.Scheme), testingutil.NewFakeClient(scheme.Scheme)).ToSyncContext("httproute-translate-test")

	hostObjects := make([]runtime.Object, 0, len(virtualHostObjects))
	for _, obj := range virtualHostObjects {
		clientObj, ok := obj.(ctrlclient.Object)
		if !ok {
			t.Fatalf("host object %T does not implement client.Object", obj)
		}
		hostObjects = append(hostObjects, utiltranslate.HostMetadata(clientObj, utiltranslate.Default.HostName(seedCtx, clientObj.GetName(), clientObj.GetNamespace())))
	}

	pClient := testingutil.NewFakeClient(scheme.Scheme, hostObjects...)
	vClient := testingutil.NewFakeClient(scheme.Scheme)
	return syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient).ToSyncContext("httproute-translate-test")
}

func httpRouteWithRuleFilterExtensionRef(ref gatewayv1.LocalObjectReference) *gatewayv1.HTTPRoute {
	return &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "route"},
		Spec: gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{{
				Filters: []gatewayv1.HTTPRouteFilter{{
					Type:         gatewayv1.HTTPRouteFilterExtensionRef,
					ExtensionRef: &ref,
				}},
			}},
		},
	}
}

func httpRouteWithBackendRefFilterExtensionRef(ref gatewayv1.LocalObjectReference) *gatewayv1.HTTPRoute {
	return &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "route"},
		Spec: gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{{
				BackendRefs: []gatewayv1.HTTPBackendRef{{
					BackendRef: gatewayv1.BackendRef{BackendObjectReference: gatewayv1.BackendObjectReference{Name: "backend"}},
					Filters: []gatewayv1.HTTPRouteFilter{{
						Type:         gatewayv1.HTTPRouteFilterExtensionRef,
						ExtensionRef: &ref,
					}},
				}},
			}},
		},
	}
}
