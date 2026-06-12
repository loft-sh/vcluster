package gatewayapi

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/config"
)

func TestUmbrellaEnablesGatewaysAndRoutesOnly(t *testing.T) {
	cfg := &config.VirtualClusterConfig{}
	cfg.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled = "auto"
	cfg.Sync.ToHost.GatewayAPI.Enabled = true

	if !GatewaysEnabled(cfg) {
		t.Fatalf("gatewayApi umbrella should enable tenant Gateway sync")
	}
	if !HTTPRoutesEnabled(cfg) {
		t.Fatalf("gatewayApi umbrella should enable HTTPRoute sync")
	}
	if !GatewayClassesImportEnabled(cfg) {
		t.Fatalf("gatewayApi umbrella should import host GatewayClasses; tenant Gateways are only eligible for sync when their GatewayClass is visible in the tenant cluster")
	}
	if ReferenceGrantSyncEnabled(cfg) {
		t.Fatalf("gatewayApi umbrella must not sync ReferenceGrants to the host without namespace sync")
	}
	if TLSRoutesEnabled(cfg) {
		t.Fatalf("gatewayApi umbrella must not enable experimental TLSRoute sync")
	}
	if BackendTLSPoliciesEnabled(cfg) {
		t.Fatalf("gatewayApi umbrella must not enable experimental BackendTLSPolicy sync")
	}

	cfg.Sync.ToHost.Namespaces.Enabled = true
	if !ReferenceGrantSyncEnabled(cfg) {
		t.Fatalf("gatewayApi umbrella with namespace sync should sync ReferenceGrants via auto")
	}
}

func TestGatewayClassesImportFollowsExplicitSwitchesAndUmbrella(t *testing.T) {
	cfg := &config.VirtualClusterConfig{}
	if GatewayClassesImportEnabled(cfg) {
		t.Fatalf("GatewayClass import must be disabled by default")
	}

	cfg.Sync.ToHost.GatewayAPI.Gateways.Enabled = true
	if GatewayClassesImportEnabled(cfg) {
		t.Fatalf("explicit tenant Gateway sync alone must not import host GatewayClasses")
	}

	cfg = &config.VirtualClusterConfig{}
	cfg.Sync.FromHost.GatewayClasses.Enabled = true
	if !GatewayClassesImportEnabled(cfg) {
		t.Fatalf("sync.fromHost.gatewayClasses.enabled should import host GatewayClasses")
	}

	cfg = &config.VirtualClusterConfig{}
	cfg.Sync.FromHost.Gateways.Enabled = true
	if !GatewayClassesImportEnabled(cfg) {
		t.Fatalf("sync.fromHost.gateways.enabled should import host GatewayClasses")
	}
}

func TestReferenceGrantSyncFollowsRouteFeaturesAndNamespaceSyncInAuto(t *testing.T) {
	cfg := &config.VirtualClusterConfig{}
	cfg.Sync.ToHost.Namespaces.Enabled = true
	cfg.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled = "auto"
	cfg.Sync.ToHost.GatewayAPI.Gateways.Enabled = true
	if ReferenceGrantSyncEnabled(cfg) {
		t.Fatalf("referenceGrants=auto must not follow Gateway sync alone")
	}

	cfg.Sync.ToHost.GatewayAPI.HTTPRoutes.Enabled = true
	if !ReferenceGrantSyncEnabled(cfg) {
		t.Fatalf("referenceGrants=auto should follow HTTPRoute sync when namespace sync is enabled")
	}

	cfg.Sync.ToHost.Namespaces.Enabled = false
	if ReferenceGrantSyncEnabled(cfg) {
		t.Fatalf("referenceGrants=auto must not sync to the host without namespace sync; the chart only grants read RBAC in single-namespace mode")
	}

	cfg.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled = "false"
	cfg.Sync.ToHost.Namespaces.Enabled = true
	if ReferenceGrantSyncEnabled(cfg) {
		t.Fatalf("referenceGrants=false should disable ReferenceGrant sync")
	}

	cfg.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled = "true"
	cfg.Sync.ToHost.GatewayAPI.HTTPRoutes.Enabled = false
	cfg.Sync.ToHost.Namespaces.Enabled = false
	if !ReferenceGrantSyncEnabled(cfg) {
		t.Fatalf("referenceGrants=true should enable ReferenceGrant sync regardless of namespace sync")
	}
}
