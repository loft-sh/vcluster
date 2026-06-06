package resources

import (
	"context"
	"strings"
	"testing"

	pkgconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	gatewayapiutil "github.com/loft-sh/vcluster/pkg/util/gatewayapi"
	"k8s.io/apimachinery/pkg/runtime/schema"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

func TestReferenceGrantAutoFollowsHTTPRouteMapperWithoutNamespaceSync(t *testing.T) {
	ctx := &synccontext.RegisterContext{Config: &pkgconfig.VirtualClusterConfig{}}
	ctx.Config.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled = "auto"
	ctx.Config.Sync.ToHost.GatewayAPI.HTTPRoutes.Enabled = true

	if !gatewayapiutil.ReferenceGrantsEnabled(ctx.Config) {
		t.Fatalf("referenceGrants=auto should install the tenant CRD when HTTPRoute mapper watches ReferenceGrant")
	}
}

func TestReferenceGrantAutoDoesNotFollowGatewayMapperAlone(t *testing.T) {
	ctx := &synccontext.RegisterContext{Config: &pkgconfig.VirtualClusterConfig{}}
	ctx.Config.Sync.ToHost.Namespaces.Enabled = true
	ctx.Config.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled = "auto"
	ctx.Config.Sync.ToHost.GatewayAPI.Gateways.Enabled = true

	if gatewayapiutil.ReferenceGrantsEnabled(ctx.Config) {
		t.Fatalf("referenceGrants=auto should not be enabled by Gateway mapper alone")
	}
}

func TestReferenceGrantMapperChecksHostCRDWithoutNamespaceSync(t *testing.T) {
	ctx := &synccontext.RegisterContext{Context: context.Background(), Config: &pkgconfig.VirtualClusterConfig{}}
	ctx.Config.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled = "auto"
	ctx.Config.Sync.ToHost.GatewayAPI.HTTPRoutes.Enabled = true

	_, err := CreateReferenceGrantMapper(ctx)
	if err == nil || !strings.Contains(err.Error(), "cannot check host cluster for Gateway API resource gateway.networking.k8s.io/v1beta1, Kind=ReferenceGrant") {
		t.Fatalf("expected ReferenceGrant host CRD check before tenant CRD install, got %v", err)
	}
}

func TestTLSRouteMapperKeepsOlderServedVersionForCompatibility(t *testing.T) {
	want := schema.GroupVersion{Group: gatewayv1alpha2.GroupVersion.Group, Version: gatewayv1alpha2.GroupVersion.Version}
	if got := mappings.TLSRoutes().GroupVersion(); got != want {
		t.Fatalf("expected TLSRoute mapper to keep older served version %s for Gateway API compatibility, got %s", want, got)
	}
}
