package gatewayapi

import (
	"testing"

	"github.com/loft-sh/vcluster/pkg/config"
)

func TestReferenceGrantsEnabledFollowsRouteFeaturesOnlyInAuto(t *testing.T) {
	cfg := &config.VirtualClusterConfig{}
	cfg.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled = "auto"
	cfg.Sync.ToHost.GatewayAPI.Gateways.Enabled = true
	if ReferenceGrantsEnabled(cfg) {
		t.Fatalf("referenceGrants=auto must not follow Gateway sync alone")
	}

	cfg.Sync.ToHost.GatewayAPI.HTTPRoutes.Enabled = true
	if !ReferenceGrantsEnabled(cfg) {
		t.Fatalf("referenceGrants=auto should follow HTTPRoute sync")
	}

	cfg.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled = "false"
	if ReferenceGrantsEnabled(cfg) {
		t.Fatalf("referenceGrants=false should disable ReferenceGrant sync")
	}

	cfg.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled = "true"
	cfg.Sync.ToHost.GatewayAPI.HTTPRoutes.Enabled = false
	if !ReferenceGrantsEnabled(cfg) {
		t.Fatalf("referenceGrants=true should enable ReferenceGrant sync")
	}
}
