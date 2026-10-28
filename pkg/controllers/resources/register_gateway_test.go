package resources

import (
	"testing"

	pkgconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
)

func TestGatewayClassesEnabledWhenGatewaysAreImported(t *testing.T) {
	ctx := &synccontext.RegisterContext{Config: &pkgconfig.VirtualClusterConfig{}}
	ctx.Config.Sync.FromHost.Gateways.Enabled = true

	if !gatewayClassesEnabled(ctx) {
		t.Fatalf("expected GatewayClass syncer to be enabled when importing Gateways")
	}
}

func TestUmbrellaEnablesTenantGatewaySync(t *testing.T) {
	ctx := &synccontext.RegisterContext{Config: &pkgconfig.VirtualClusterConfig{}}
	if gatewayGatewaysEnabled(ctx) {
		t.Fatalf("tenant Gateway sync must be disabled by default")
	}

	ctx.Config.Sync.ToHost.GatewayAPI.Enabled = true
	if !gatewayGatewaysEnabled(ctx) {
		t.Fatalf("expected gatewayApi umbrella to enable tenant Gateway sync")
	}

	ctx.Config.Sync.ToHost.GatewayAPI.Enabled = false
	ctx.Config.Sync.ToHost.GatewayAPI.Gateways.Enabled = true
	if !gatewayGatewaysEnabled(ctx) {
		t.Fatalf("expected explicit gatewayApi.gateways.enabled to enable tenant Gateway sync")
	}
}

func TestGatewayClassSyncNotEnabledByTenantGatewaySyncAlone(t *testing.T) {
	ctx := &synccontext.RegisterContext{Config: &pkgconfig.VirtualClusterConfig{}}
	ctx.Config.Sync.ToHost.GatewayAPI.Gateways.Enabled = true

	if gatewayClassesEnabled(ctx) {
		t.Fatalf("tenant Gateway sync should not enable host GatewayClass sync by itself")
	}
}

func TestGatewayClassSyncEnabledByUmbrella(t *testing.T) {
	ctx := &synccontext.RegisterContext{Config: &pkgconfig.VirtualClusterConfig{}}
	ctx.Config.Sync.ToHost.GatewayAPI.Enabled = true

	if !gatewayClassesEnabled(ctx) {
		t.Fatalf("gatewayApi umbrella should enable host GatewayClass sync; tenant Gateways are only eligible when their GatewayClass is visible in the tenant cluster")
	}
}

func TestReferenceGrantAutoDoesNotFollowGatewaySyncAlone(t *testing.T) {
	ctx := &synccontext.RegisterContext{Config: &pkgconfig.VirtualClusterConfig{}}
	ctx.Config.Sync.ToHost.Namespaces.Enabled = true
	ctx.Config.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled = "auto"
	ctx.Config.Sync.ToHost.GatewayAPI.Gateways.Enabled = true

	if gatewayReferenceGrantsEnabled(ctx) {
		t.Fatalf("referenceGrants=auto should not be enabled by Gateway sync alone")
	}
}

func TestReferenceGrantSyncerRequiresNamespaceSyncInAuto(t *testing.T) {
	ctx := &synccontext.RegisterContext{Config: &pkgconfig.VirtualClusterConfig{}}
	ctx.Config.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled = "auto"
	ctx.Config.Sync.ToHost.GatewayAPI.HTTPRoutes.Enabled = true

	if gatewayReferenceGrantsEnabled(ctx) {
		t.Fatalf("referenceGrants=auto must not start the host syncer without namespace sync; the chart only grants read RBAC in single-namespace mode")
	}

	ctx.Config.Sync.ToHost.Namespaces.Enabled = true
	if !gatewayReferenceGrantsEnabled(ctx) {
		t.Fatalf("referenceGrants=auto should start the host syncer when HTTPRoute and namespace sync are enabled")
	}

	ctx.Config.Sync.ToHost.Namespaces.Enabled = false
	ctx.Config.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled = "true"
	if !gatewayReferenceGrantsEnabled(ctx) {
		t.Fatalf("referenceGrants=true should start the host syncer regardless of namespace sync")
	}
}
