package resources

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
)

// ExtraMappers that will be started as well
var ExtraMappers []BuildMapper

// BuildMapper is a function to build a new mapper
type BuildMapper func(ctx *synccontext.RegisterContext) (synccontext.Mapper, error)

func getMappers(ctx *synccontext.RegisterContext) []BuildMapper {
	return append([]BuildMapper{
		CreateSecretsMapper,
		CreateConfigMapsMapper,
		CreateEndpointsMapper,
		CreateEndpointSlicesMapper,
		CreateEventsMapper,
		isEnabled(ctx.Config.Sync.ToHost.Ingresses.Enabled, CreateIngressesMapper),
		isEnabled(gatewayGatewaysEnabled(ctx) || ctx.Config.Sync.FromHost.Gateways.Enabled, CreateGatewayMapper),
		isEnabled(gatewayHTTPRoutesEnabled(ctx), CreateHTTPRouteMapper),
		isEnabled(gatewayTLSRoutesEnabled(ctx), CreateTLSRouteMapper),
		isEnabled(gatewayBackendTLSPoliciesEnabled(ctx), CreateBackendTLSPolicyMapper),
		isEnabled(gatewayReferenceGrantsEnabled(ctx), CreateReferenceGrantMapper),
		CreateNamespacesMapper,
		isEnabled(ctx.Config.Sync.ToHost.NetworkPolicies.Enabled, CreateNetworkPoliciesMapper),
		CreateNodesMapper,
		CreatePersistentVolumeClaimsMapper,
		isEnabled(ctx.Config.Sync.ToHost.ServiceAccounts.Enabled, CreateServiceAccountsMapper),
		CreateServiceMapper,
		isEnabled(ctx.Config.Sync.ToHost.PriorityClasses.Enabled || ctx.Config.Sync.FromHost.PriorityClasses.Enabled, CreatePriorityClassesMapper),
		CreatePersistentVolumesMapper,
		CreatePodsMapper,
		CreateStorageClassesMapper,
		CreateVolumeSnapshotClassesMapper,
		CreateVolumeSnapshotContentsMapper,
		CreateVolumeSnapshotsMapper,
		isEnabled(ctx.Config.Sync.ToHost.ResourceClaims.Enabled, CreateResourceClaimsMapper),
		isEnabled(ctx.Config.Sync.ToHost.ResourceClaimTemplates.Enabled, CreateResourceClaimTemplatesMapper),
		isEnabled(ctx.Config.Sync.FromHost.DeviceClasses.Enabled, CreateDeviceClassesMapper),
	}, ExtraMappers...)
}

func MustRegisterMappings(ctx *synccontext.RegisterContext) {
	err := RegisterMappings(ctx)
	if err != nil {
		panic(err.Error())
	}
}

func RegisterMappings(ctx *synccontext.RegisterContext) error {
	// create mappers
	for _, createFunc := range getMappers(ctx) {
		if createFunc == nil {
			continue
		}

		mapper, err := createFunc(ctx)
		if err != nil {
			return fmt.Errorf("create mapper: %w", err)
		} else if mapper == nil {
			continue
		}

		err = ctx.Mappings.AddMapper(mapper)
		if err != nil {
			return fmt.Errorf("add mapper %s: %w", mapper.GroupVersionKind().String(), err)
		}
	}

	return nil
}

func isEnabled[T any](enabled bool, fn T) T {
	if enabled {
		return fn
	}
	var ret T
	return ret
}

func gatewayGatewaysEnabled(ctx *synccontext.RegisterContext) bool {
	return ctx.Config.Sync.ToHost.GatewayAPI.Gateways.Enabled
}

func gatewayHTTPRoutesEnabled(ctx *synccontext.RegisterContext) bool {
	return ctx.Config.Sync.ToHost.GatewayAPI.HTTPRoutes.Enabled || ctx.Config.Sync.ToHost.GatewayAPI.Enabled
}

func gatewayTLSRoutesEnabled(ctx *synccontext.RegisterContext) bool {
	return ctx.Config.Sync.ToHost.GatewayAPI.TLSRoutes.Enabled
}

func gatewayBackendTLSPoliciesEnabled(ctx *synccontext.RegisterContext) bool {
	return ctx.Config.Sync.ToHost.GatewayAPI.BackendTLSPolicies.Enabled
}

func gatewayReferenceGrantsEnabled(ctx *synccontext.RegisterContext) bool {
	mode := ctx.Config.Sync.ToHost.GatewayAPI.ReferenceGrants.Enabled
	if mode == "true" {
		return true
	}
	if mode == "false" {
		return false
	}
	return ctx.Config.Sync.ToHost.Namespaces.Enabled && (gatewayHTTPRoutesEnabled(ctx) || gatewayTLSRoutesEnabled(ctx) || gatewayBackendTLSPoliciesEnabled(ctx))
}
