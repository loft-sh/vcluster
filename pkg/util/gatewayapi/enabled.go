package gatewayapi

import "github.com/loft-sh/vcluster/pkg/config"

// GatewaysEnabled reports whether tenant-created Gateway sync is enabled,
// either explicitly or via the gatewayApi umbrella switch.
func GatewaysEnabled(config *config.VirtualClusterConfig) bool {
	return config.Sync.ToHost.GatewayAPI.Gateways.Enabled || config.Sync.ToHost.GatewayAPI.Enabled
}

// GatewayClassesImportEnabled reports whether host GatewayClasses are imported
// into the tenant cluster. The gatewayApi umbrella switch implies the import:
// tenant Gateway sync only accepts Gateways whose GatewayClass is visible in
// the tenant cluster, so umbrella-only installs would otherwise sync nothing.
// The explicit sync.toHost.gatewayApi.gateways switch does not imply it; there
// the user opts into each piece individually.
func GatewayClassesImportEnabled(config *config.VirtualClusterConfig) bool {
	return config.Sync.FromHost.GatewayClasses.Enabled || config.Sync.FromHost.Gateways.Enabled || config.Sync.ToHost.GatewayAPI.Enabled
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

// ReferenceGrantSyncEnabled reports whether tenant ReferenceGrants sync to the
// host cluster. In auto mode grants follow route/policy sync but additionally
// require namespace sync: without it all tenant namespaces collapse into one
// host namespace, so grants are only validated against virtual objects and the
// chart grants read-only host RBAC for them. The tenant ReferenceGrant CRD is
// installed independently by the route mappers (EnsureReferenceGrantCRD).
func ReferenceGrantSyncEnabled(config *config.VirtualClusterConfig) bool {
	mode := config.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled
	if mode == "true" {
		return true
	}
	if mode == "false" {
		return false
	}
	routesEnabled := HTTPRoutesEnabled(config) || TLSRoutesEnabled(config) || BackendTLSPoliciesEnabled(config)
	return routesEnabled && config.Sync.ToHost.Namespaces.Enabled
}
