package authz

import (
	"context"
	"testing"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	utiltranslate "github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestHTTPRouteAttachmentRespectsAllowedRouteNamespaces(t *testing.T) {
	fromSame := gatewayv1.NamespacesFromSame
	fromAll := gatewayv1.NamespacesFromAll
	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "gateways", Name: "edge"},
		Spec: gatewayv1.GatewaySpec{Listeners: []gatewayv1.Listener{
			{Name: "same", Port: 80, Protocol: gatewayv1.HTTPProtocolType, AllowedRoutes: &gatewayv1.AllowedRoutes{Namespaces: &gatewayv1.RouteNamespaces{From: &fromSame}}},
			{Name: "all", Port: 8080, Protocol: gatewayv1.HTTPProtocolType, AllowedRoutes: &gatewayv1.AllowedRoutes{Namespaces: &gatewayv1.RouteNamespaces{From: &fromAll}}},
		}},
	}
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: &config.VirtualClusterConfig{}, VirtualClient: testingutil.NewFakeClient(scheme.Scheme, gateway)}

	if err := HTTPRouteAttachment(ctx, "gateways", &gatewayv1.ParentReference{Name: "edge", SectionName: ptr.To(gatewayv1.SectionName("same"))}); err != nil {
		t.Fatalf("expected same-namespace route attachment to be allowed: %v", err)
	}

	err := HTTPRouteAttachment(ctx, "tenant-a", &gatewayv1.ParentReference{Name: "edge", Namespace: ptr.To(gatewayv1.Namespace("gateways")), SectionName: ptr.To(gatewayv1.SectionName("same"))})
	if !IsNotPermitted(err) {
		t.Fatalf("expected cross-namespace attachment to same-only listener to be denied, got %v", err)
	}

	if err := HTTPRouteAttachment(ctx, "tenant-a", &gatewayv1.ParentReference{Name: "edge", Namespace: ptr.To(gatewayv1.Namespace("gateways")), SectionName: ptr.To(gatewayv1.SectionName("all"))}); err != nil {
		t.Fatalf("expected all-namespaces listener to allow cross-namespace route: %v", err)
	}
}

func TestHTTPRouteAttachmentRespectsAllowedRouteNamespaceSelector(t *testing.T) {
	fromSelector := gatewayv1.NamespacesFromSelector
	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "gateways", Name: "edge"},
		Spec: gatewayv1.GatewaySpec{Listeners: []gatewayv1.Listener{{
			Name:     "http",
			Port:     80,
			Protocol: gatewayv1.HTTPProtocolType,
			AllowedRoutes: &gatewayv1.AllowedRoutes{Namespaces: &gatewayv1.RouteNamespaces{
				From:     &fromSelector,
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"team": "a"}},
			}},
		}}},
	}
	teamA := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "team-a", Labels: map[string]string{"team": "a"}}}
	teamB := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "team-b", Labels: map[string]string{"team": "b"}}}
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: &config.VirtualClusterConfig{}, VirtualClient: testingutil.NewFakeClient(scheme.Scheme, gateway, teamA, teamB)}
	parent := gatewayv1.ParentReference{Name: "edge", Namespace: ptr.To(gatewayv1.Namespace("gateways"))}

	if err := HTTPRouteAttachment(ctx, "team-a", &parent); err != nil {
		t.Fatalf("expected selected namespace to attach: %v", err)
	}
	if err := HTTPRouteAttachment(ctx, "team-b", &parent); !IsNotPermitted(err) {
		t.Fatalf("expected unselected namespace attachment to be denied, got %v", err)
	}
}

func TestHTTPRouteBackendRequiresReferenceGrantForCrossNamespaceRefsInSingleNamespaceMode(t *testing.T) {
	restore := setDefaultTranslator(utiltranslate.NewSingleNamespaceTranslator("vcluster-host"))
	defer restore()

	ctx := &synccontext.SyncContext{Context: context.Background(), Config: &config.VirtualClusterConfig{}, VirtualClient: testingutil.NewFakeClient(scheme.Scheme)}
	backendNamespace := gatewayv1.Namespace("backends")
	backend := gatewayv1.BackendObjectReference{Name: "api", Namespace: &backendNamespace}

	if err := HTTPRouteBackend(ctx, "routes", &backend); !IsNotPermitted(err) {
		t.Fatalf("expected cross-namespace backend reference without ReferenceGrant to be denied, got %v", err)
	}

	grant := &gatewayv1.ReferenceGrant{
		ObjectMeta: metav1.ObjectMeta{Namespace: "backends", Name: "allow-routes"},
		Spec: gatewayv1.ReferenceGrantSpec{
			From: []gatewayv1.ReferenceGrantFrom{{Group: gatewayv1.Group(gatewayv1.GroupVersion.Group), Kind: gatewayv1.Kind("HTTPRoute"), Namespace: gatewayv1.Namespace("routes")}},
			To:   []gatewayv1.ReferenceGrantTo{{Group: gatewayv1.Group(corev1.GroupName), Kind: gatewayv1.Kind("Service"), Name: ptr.To(gatewayv1.ObjectName("api"))}},
		},
	}
	if err := ctx.VirtualClient.Create(context.Background(), grant); err != nil {
		t.Fatalf("create ReferenceGrant: %v", err)
	}
	if err := HTTPRouteBackend(ctx, "routes", &backend); err != nil {
		t.Fatalf("expected matching ReferenceGrant to allow backend reference: %v", err)
	}
}

func setDefaultTranslator(translator utiltranslate.Translator) func() {
	previous := utiltranslate.Default
	utiltranslate.Default = translator
	return func() { utiltranslate.Default = previous }
}

func TestHTTPRouteAttachmentDeniesMissingGateway(t *testing.T) {
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: &config.VirtualClusterConfig{}, VirtualClient: testingutil.NewFakeClient(scheme.Scheme)}
	err := HTTPRouteAttachment(ctx, "routes", &gatewayv1.ParentReference{Name: "missing", Namespace: ptr.To(gatewayv1.Namespace("gateways"))})
	if !IsNotPermitted(err) {
		t.Fatalf("expected missing Gateway attachment to be denied, got %v", err)
	}
}

func TestHTTPRouteAttachmentIgnoresServiceParentRefs(t *testing.T) {
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: &config.VirtualClusterConfig{}, VirtualClient: testingutil.NewFakeClient(scheme.Scheme)}
	ref := &gatewayv1.ParentReference{Group: ptr.To(gatewayv1.Group(corev1.GroupName)), Kind: ptr.To(gatewayv1.Kind("Service")), Name: "svc", Namespace: ptr.To(gatewayv1.Namespace("other"))}
	if err := HTTPRouteAttachment(ctx, "routes", ref); err != nil {
		t.Fatalf("expected Service parentRefs to bypass Gateway attachment authorization: %v", err)
	}
}

func TestNamespaceSelectorMissingNamespaceIsDeniedNotFatal(t *testing.T) {
	fromSelector := gatewayv1.NamespacesFromSelector
	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "gateways", Name: "edge"},
		Spec: gatewayv1.GatewaySpec{Listeners: []gatewayv1.Listener{{
			Name: "http", Port: 80, Protocol: gatewayv1.HTTPProtocolType,
			AllowedRoutes: &gatewayv1.AllowedRoutes{Namespaces: &gatewayv1.RouteNamespaces{From: &fromSelector, Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"team": "a"}}}},
		}}},
	}
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: &config.VirtualClusterConfig{}, VirtualClient: testingutil.NewFakeClient(scheme.Scheme, gateway)}
	err := HTTPRouteAttachment(ctx, "missing-ns", &gatewayv1.ParentReference{Name: "edge", Namespace: ptr.To(gatewayv1.Namespace("gateways"))})
	if !IsNotPermitted(err) {
		t.Fatalf("expected route in missing namespace to be denied, got %v", err)
	}
}

func TestReferenceGrantNameOmittedAllowsAnyTargetName(t *testing.T) {
	restore := setDefaultTranslator(utiltranslate.NewSingleNamespaceTranslator("vcluster-host"))
	defer restore()

	grant := &gatewayv1.ReferenceGrant{
		ObjectMeta: metav1.ObjectMeta{Namespace: "backends", Name: "allow-any-service"},
		Spec: gatewayv1.ReferenceGrantSpec{
			From: []gatewayv1.ReferenceGrantFrom{{Group: gatewayv1.Group(gatewayv1.GroupVersion.Group), Kind: gatewayv1.Kind("HTTPRoute"), Namespace: gatewayv1.Namespace("routes")}},
			To:   []gatewayv1.ReferenceGrantTo{{Group: gatewayv1.Group(corev1.GroupName), Kind: gatewayv1.Kind("Service")}},
		},
	}
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: &config.VirtualClusterConfig{}, VirtualClient: testingutil.NewFakeClient(scheme.Scheme, grant)}
	backendNamespace := gatewayv1.Namespace("backends")

	if err := HTTPRouteBackend(ctx, "routes", &gatewayv1.BackendObjectReference{Name: "any-service", Namespace: &backendNamespace}); err != nil {
		t.Fatalf("expected ReferenceGrant without to.name to allow any target Service name: %v", err)
	}
}

func TestHTTPRouteAttachmentSectionAndPortMustSelectACompatibleListener(t *testing.T) {
	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "gateways", Name: "edge"},
		Spec:       gatewayv1.GatewaySpec{Listeners: []gatewayv1.Listener{{Name: "https", Port: 443, Protocol: gatewayv1.HTTPSProtocolType}}},
	}
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: &config.VirtualClusterConfig{}, VirtualClient: testingutil.NewFakeClient(scheme.Scheme, gateway)}

	err := HTTPRouteAttachment(ctx, "gateways", &gatewayv1.ParentReference{Name: "edge", SectionName: ptr.To(gatewayv1.SectionName("missing"))})
	if !IsNotPermitted(err) {
		t.Fatalf("expected non-matching sectionName to be denied, got %v", err)
	}

	err = HTTPRouteAttachment(ctx, "gateways", &gatewayv1.ParentReference{Name: "edge", Port: ptr.To[int32](80)})
	if !IsNotPermitted(err) {
		t.Fatalf("expected non-matching port to be denied, got %v", err)
	}

	if err := HTTPRouteAttachment(ctx, "gateways", &gatewayv1.ParentReference{Name: "edge", Port: ptr.To[int32](443)}); err != nil {
		t.Fatalf("expected matching HTTPS listener to accept HTTPRoute: %v", err)
	}
}

func TestReferenceGrantRequiresMatchingFromNamespace(t *testing.T) {
	restore := setDefaultTranslator(utiltranslate.NewSingleNamespaceTranslator("vcluster-host"))
	defer restore()

	grant := &gatewayv1.ReferenceGrant{
		ObjectMeta: metav1.ObjectMeta{Namespace: "backends", Name: "allow-other"},
		Spec: gatewayv1.ReferenceGrantSpec{
			From: []gatewayv1.ReferenceGrantFrom{{Group: gatewayv1.Group(gatewayv1.GroupVersion.Group), Kind: gatewayv1.Kind("HTTPRoute"), Namespace: gatewayv1.Namespace("other-routes")}},
			To:   []gatewayv1.ReferenceGrantTo{{Group: gatewayv1.Group(corev1.GroupName), Kind: gatewayv1.Kind("Service")}},
		},
	}
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: &config.VirtualClusterConfig{}, VirtualClient: testingutil.NewFakeClient(scheme.Scheme, grant)}
	backendNamespace := gatewayv1.Namespace("backends")
	err := HTTPRouteBackend(ctx, "routes", &gatewayv1.BackendObjectReference{Name: "api", Namespace: &backendNamespace})
	if !IsNotPermitted(err) {
		t.Fatalf("expected ReferenceGrant with mismatched from namespace to deny reference, got %v", err)
	}
}

func TestReferenceGrantTargetNamespaceDefaultsToLocalNamespace(t *testing.T) {
	restore := setDefaultTranslator(utiltranslate.NewSingleNamespaceTranslator("vcluster-host"))
	defer restore()

	ctx := &synccontext.SyncContext{Context: context.Background(), Config: &config.VirtualClusterConfig{}, VirtualClient: testingutil.NewFakeClient(scheme.Scheme)}
	if err := HTTPRouteBackend(ctx, "routes", &gatewayv1.BackendObjectReference{Name: "api"}); err != nil {
		t.Fatalf("expected same-namespace backend ref to be allowed without ReferenceGrant: %v", err)
	}
}
