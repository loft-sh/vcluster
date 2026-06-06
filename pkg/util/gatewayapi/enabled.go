package gatewayapi

import "github.com/loft-sh/vcluster/pkg/config"

// GatewaysEnabled reports whether tenant-created Gateway sync is explicitly enabled.
func GatewaysEnabled(config *config.VirtualClusterConfig) bool {
	return config.Sync.ToHost.GatewayAPI.Gateways.Enabled
}

// HTTPRoutesEnabled reports whether HTTPRoute sync is enabled via the legacy umbrella or explicit switch.
func HTTPRoutesEnabled(config *config.VirtualClusterConfig) bool {
	return config.Sync.ToHost.GatewayAPI.HTTPRoutes.Enabled || config.Sync.ToHost.GatewayAPI.Enabled
}

// TLSRoutesEnabled reports whether TLSRoute sync is enabled.
func TLSRoutesEnabled(config *config.VirtualClusterConfig) bool {
	return config.Sync.ToHost.GatewayAPI.TLSRoutes.Enabled
}

// BackendTLSPoliciesEnabled reports whether BackendTLSPolicy sync is enabled.
func BackendTLSPoliciesEnabled(config *config.VirtualClusterConfig) bool {
	return config.Sync.ToHost.GatewayAPI.BackendTLSPolicies.Enabled
}

// ReferenceGrantsEnabled reports whether ReferenceGrant sync/CRD setup is required.
func ReferenceGrantsEnabled(config *config.VirtualClusterConfig) bool {
	mode := config.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled
	if mode == "true" {
		return true
	}
	if mode == "false" {
		return false
	}
	return HTTPRoutesEnabled(config) || TLSRoutesEnabled(config) || BackendTLSPoliciesEnabled(config)
}
