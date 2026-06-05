package resources

import (
	"testing"

	pkgconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"k8s.io/apimachinery/pkg/runtime/schema"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

func TestReferenceGrantAutoFollowsHTTPRouteMapperWithoutNamespaceSync(t *testing.T) {
	ctx := &synccontext.RegisterContext{Config: &pkgconfig.VirtualClusterConfig{}}
	ctx.Config.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled = "auto"
	ctx.Config.Sync.ToHost.GatewayAPI.HTTPRoutes.Enabled = true

	if !gatewayReferenceGrantsEnabled(ctx) {
		t.Fatalf("referenceGrants=auto should install the tenant CRD when HTTPRoute mapper watches ReferenceGrant")
	}
}

func TestReferenceGrantAutoDoesNotFollowGatewayMapperAlone(t *testing.T) {
	ctx := &synccontext.RegisterContext{Config: &pkgconfig.VirtualClusterConfig{}}
	ctx.Config.Sync.ToHost.Namespaces.Enabled = true
	ctx.Config.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled = "auto"
	ctx.Config.Sync.ToHost.GatewayAPI.Gateways.Enabled = true

	if gatewayReferenceGrantsEnabled(ctx) {
		t.Fatalf("referenceGrants=auto should not be enabled by Gateway mapper alone")
	}
}

func TestTLSRouteMapperKeepsOlderServedVersionForCompatibility(t *testing.T) {
	want := schema.GroupVersion{Group: gatewayv1alpha2.GroupVersion.Group, Version: gatewayv1alpha2.GroupVersion.Version}
	if got := mappings.TLSRoutes().GroupVersion(); got != want {
		t.Fatalf("expected TLSRoute mapper to keep older served version %s for Gateway API compatibility, got %s", want, got)
	}
}
