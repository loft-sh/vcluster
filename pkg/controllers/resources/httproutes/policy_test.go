package httproutes

import (
	"context"
	"testing"

	rootconfig "github.com/loft-sh/vcluster/config"
	pkgconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertesting "github.com/loft-sh/vcluster/pkg/syncer/testing"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestValidateImportedGatewayHostnamePolicyRejectsDisallowedHostnameThroughMapping(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Mappings.ByName = map[string]string{"networking/shared-edge": "tenant-gateways/edge"}
	vcConfig.Sync.FromHost.Gateways.AllowedRoutes.Overrides = []rootconfig.GatewayAllowedRoutesPolicyOverride{{
		HostNamespace:    "networking",
		Name:             "shared-edge",
		AllowedHostnames: []string{"*.team-a.example.com"},
	}}
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: vcConfig}
	parentNamespace := gatewayv1.Namespace("tenant-gateways")
	route := &gatewayv1.HTTPRoute{}
	route.Namespace = "demo"
	route.Spec.ParentRefs = []gatewayv1.ParentReference{{Name: "edge", Namespace: &parentNamespace}}
	route.Spec.Hostnames = []gatewayv1.Hostname{"admin.example.com"}

	if err := validateImportedGatewayHostnamePolicy(ctx, route); err == nil {
		t.Fatalf("expected disallowed hostname to be rejected")
	}
}

func TestValidateImportedGatewayHostnamePolicyAllowsWildcardMatchThroughMapping(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Mappings.ByName = map[string]string{"networking/shared-edge": "tenant-gateways/edge"}
	vcConfig.Sync.FromHost.Gateways.AllowedRoutes.Overrides = []rootconfig.GatewayAllowedRoutesPolicyOverride{{
		HostNamespace:    "networking",
		Name:             "shared-edge",
		AllowedHostnames: []string{"*.team-a.example.com"},
	}}
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: vcConfig}
	parentNamespace := gatewayv1.Namespace("tenant-gateways")
	route := &gatewayv1.HTTPRoute{}
	route.Namespace = "demo"
	route.Spec.ParentRefs = []gatewayv1.ParentReference{{Name: "edge", Namespace: &parentNamespace}}
	route.Spec.Hostnames = []gatewayv1.Hostname{"api.team-a.example.com"}

	if err := validateImportedGatewayHostnamePolicy(ctx, route); err != nil {
		t.Fatalf("expected wildcard hostname to be allowed: %v", err)
	}
}

func TestValidateImportedGatewayHostnamePolicyIgnoresUnmappedParent(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.FromHost.Gateways.Mappings.ByName = map[string]string{"networking/shared-edge": "tenant-gateways/edge"}
	vcConfig.Sync.FromHost.Gateways.AllowedRoutes.Overrides = []rootconfig.GatewayAllowedRoutesPolicyOverride{{
		HostNamespace:    "networking",
		Name:             "shared-edge",
		AllowedHostnames: []string{"*.team-a.example.com"},
	}}
	ctx := &synccontext.SyncContext{Context: context.Background(), Config: vcConfig}
	parentNamespace := gatewayv1.Namespace("team-a")
	route := &gatewayv1.HTTPRoute{}
	route.Namespace = "demo"
	route.Spec.ParentRefs = []gatewayv1.ParentReference{{Name: "tenant-gateway", Namespace: &parentNamespace}}
	route.Spec.Hostnames = []gatewayv1.Hostname{"admin.example.com"}

	if err := validateImportedGatewayHostnamePolicy(ctx, route); err != nil {
		t.Fatalf("expected unmapped parent to ignore imported Gateway hostname policy: %v", err)
	}
}

func TestHTTPRouteParentRefCanTargetManagedTenantGateway(t *testing.T) {
	vcConfig := &pkgconfig.VirtualClusterConfig{}
	vcConfig.Sync.ToHost.GatewayAPI.Gateways.Enabled = true
	vcConfig.Sync.ToHost.GatewayAPI.HTTPRoutes.Enabled = true
	fromAll := gatewayv1.NamespacesFromAll
	virtualGateway := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "edge"}, Spec: gatewayv1.GatewaySpec{Listeners: []gatewayv1.Listener{{Name: "http", Port: 80, Protocol: gatewayv1.HTTPProtocolType, AllowedRoutes: &gatewayv1.AllowedRoutes{Namespaces: &gatewayv1.RouteNamespaces{From: &fromAll}}}}}}
	pClient := testingutil.NewFakeClient(scheme.Scheme)
	vClient := testingutil.NewFakeClient(scheme.Scheme, virtualGateway)
	registerCtx := syncertesting.NewFakeRegisterContext(vcConfig, pClient, vClient)
	syncCtx := registerCtx.ToSyncContext("httproute-test")
	hostGatewayName := translate.Default.HostName(syncCtx, "edge", "team-a")
	hostGateway := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: hostGatewayName.Namespace, Name: hostGatewayName.Name, Labels: map[string]string{translate.MarkerLabel: translate.VClusterName}, Annotations: map[string]string{translate.NameAnnotation: "edge", translate.NamespaceAnnotation: "team-a"}}}
	if err := pClient.Create(context.Background(), hostGateway); err != nil {
		t.Fatalf("create host Gateway fixture: %v", err)
	}

	route := &gatewayv1.HTTPRoute{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "app"}, Spec: gatewayv1.HTTPRouteSpec{CommonRouteSpec: gatewayv1.CommonRouteSpec{ParentRefs: []gatewayv1.ParentReference{{Name: "edge"}}}}}
	spec, err := specToHost(syncCtx, route, true)
	if err != nil {
		t.Fatalf("expected tenant-created managed Gateway parentRef to translate: %v", err)
	}
	if spec.ParentRefs[0].Name != gatewayv1.ObjectName(hostGatewayName.Name) {
		t.Fatalf("expected parentRef to translate to host Gateway name %q, got %q", hostGatewayName.Name, spec.ParentRefs[0].Name)
	}
}
