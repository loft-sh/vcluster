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
