package registermappings

import (
	"fmt"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/resources"
)

type CreateMapper func(ctx *synccontext.RegisterContext) (mappings.Mapper, error)

var DefaultResourceMappings = []CreateMapper{
	resources.CreateSecretsMapper,
	resources.CreateConfigMapsMapper,
	resources.CreateCSIDriversMapper,
	resources.CreateCSINodesMapper,
	resources.CreateCSIStorageCapacitiesMapper,
	resources.CreateEndpointsMapper,
	resources.CreateEventsMapper,
	resources.CreateIngressClassesMapper,
	resources.CreateIngressesMapper,
	resources.CreateNamespacesMapper,
	resources.CreateNetworkPoliciesMapper,
	resources.CreateNodesMapper,
	resources.CreatePersistentVolumeClaimsMapper,
	resources.CreateServiceAccountsMapper,
	resources.CreateServiceMapper,
	resources.CreatePriorityClassesMapper,
	resources.CreatePodDisruptionBudgetsMapper,
	resources.CreatePersistentVolumesMapper,
	resources.CreatePodsMapper,
	resources.CreateStorageClassesMapper,
	resources.CreateVolumeSnapshotClassesMapper,
	resources.CreateVolumeSnapshotContentsMapper,
	resources.CreateVolumeSnapshotsMapper,
}

func MustRegisterMappings(ctx *synccontext.RegisterContext) {
	err := RegisterMappings(ctx)
	if err != nil {
		panic(err.Error())
	}
}

func RegisterMappings(ctx *synccontext.RegisterContext) error {
	// create mappers
	for _, createFunc := range DefaultResourceMappings {
		if createFunc == nil {
			continue
		}

		mapper, err := createFunc(ctx)
		if err != nil {
			return fmt.Errorf("create mapper: %w", err)
		}

		err = mappings.Default.AddMapper(mapper)
		if err != nil {
			return fmt.Errorf("add mapper %s: %w", mapper.GroupVersionKind().String(), err)
		}
	}

	return nil
}
