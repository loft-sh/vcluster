package resources

import (
	"context"
	"strings"
	"testing"

	pkgconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util"
	gatewayapiutil "github.com/loft-sh/vcluster/pkg/util/gatewayapi"
	testingutil "github.com/loft-sh/vcluster/pkg/util/testing"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
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

func TestRouteMappersEnsureReferenceGrantCRDWhenGrantSyncDisabled(t *testing.T) {
	ensured := map[schema.GroupVersionKind]bool{}
	restoreEnsureCRD := util.EnsureCRD
	restoreKindExists := util.KindExists
	util.EnsureCRD = func(_ context.Context, _ *rest.Config, _ []byte, gvk schema.GroupVersionKind) error {
		ensured[gvk] = true
		return nil
	}
	util.KindExists = func(_ *rest.Config, _ schema.GroupVersionKind) (bool, error) {
		return true, nil
	}
	t.Cleanup(func() {
		util.EnsureCRD = restoreEnsureCRD
		util.KindExists = restoreKindExists
	})

	fakeClient := testingutil.NewFakeClient(scheme.Scheme)
	ctx := &synccontext.RegisterContext{
		Context:        context.Background(),
		Config:         &pkgconfig.VirtualClusterConfig{},
		VirtualManager: testingutil.NewFakeManager(fakeClient),
		HostManager:    testingutil.NewFakeManager(fakeClient),
	}
	ctx.Config.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled = "false"
	ctx.Config.Sync.ToHost.GatewayAPI.HTTPRoutes.Enabled = true
	ctx.Config.Sync.ToHost.GatewayAPI.TLSRoutes.Enabled = true

	if _, err := CreateHTTPRouteMapper(ctx); err != nil {
		t.Fatalf("create HTTPRoute mapper: %v", err)
	}
	if _, err := CreateTLSRouteMapper(ctx); err != nil {
		t.Fatalf("create TLSRoute mapper: %v", err)
	}

	if !ensured[mappings.ReferenceGrants()] {
		t.Fatalf("route mappers must ensure the tenant ReferenceGrant CRD even with grant sync disabled; route controllers watch virtual ReferenceGrants for cross-namespace authorization")
	}
}

func TestTLSRouteMapperKeepsOlderServedVersionForCompatibility(t *testing.T) {
	want := schema.GroupVersion{Group: gatewayv1alpha2.GroupVersion.Group, Version: gatewayv1alpha2.GroupVersion.Version}
	if got := mappings.TLSRoutes().GroupVersion(); got != want {
		t.Fatalf("expected TLSRoute mapper to keep older served version %s for Gateway API compatibility, got %s", want, got)
	}
}
