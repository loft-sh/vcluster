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

func TestTenantGatewaysRequireExplicitToggle(t *testing.T) {
	ctx := &synccontext.RegisterContext{Config: &pkgconfig.VirtualClusterConfig{}}
	ctx.Config.Sync.ToHost.GatewayAPI.Enabled = true
	if gatewayGatewaysEnabled(ctx) {
		t.Fatalf("legacy gatewayApi umbrella must not enable tenant Gateway sync")
	}

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

func TestReferenceGrantAutoDoesNotFollowGatewaySyncAlone(t *testing.T) {
	ctx := &synccontext.RegisterContext{Config: &pkgconfig.VirtualClusterConfig{}}
	ctx.Config.Sync.ToHost.Namespaces.Enabled = true
	ctx.Config.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled = "auto"
	ctx.Config.Sync.ToHost.GatewayAPI.Gateways.Enabled = true

	if gatewayReferenceGrantsEnabled(ctx) {
		t.Fatalf("referenceGrants=auto should not be enabled by Gateway sync alone")
	}
}
