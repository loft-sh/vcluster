package registermappings

import (
	"fmt"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings/resources"
)

type registerMapping func(ctx *synccontext.RegisterContext) error

var mappings = []registerMapping{
	resources.RegisterSecretsMapper,
	resources.RegisterConfigMapsMapper,
	resources.RegisterCSIDriversMapper,
	resources.RegisterCSINodesMapper,
	resources.RegisterCSIStorageCapacitiesMapper,
	resources.RegisterEndpointsMapper,
	resources.RegisterEventsMapper,
	resources.RegisterIngressClassesMapper,
	resources.RegisterIngressesMapper,
	resources.RegisterIngressesLegacyMapper,
	resources.RegisterNamespacesMapper,
	resources.RegisterNetworkPoliciesMapper,
	resources.RegisterNodesMapper,
	resources.RegisterPersistentVolumeClaimsMapper,
	resources.RegisterServiceAccountsMapper,
	resources.RegisterServiceMapper,
	resources.RegisterPriorityClassesMapper,
	resources.RegisterPodDisruptionBudgetsMapper,
	resources.RegisterPersistentVolumesMapper,
	resources.RegisterPodsMapper,
	resources.RegisterStorageClassesMapper,
	resources.RegisterVolumeSnapshotClassesMapper,
	resources.RegisterVolumeSnapshotContentsMapper,
	resources.RegisterVolumeSnapshotsMapper,
	resources.RegisterGenericExporterMappers,
}

func RegisterMappings(ctx *synccontext.RegisterContext) error {
	for _, register := range mappings {
		if register == nil {
			continue
		}

		err := register(ctx)
		if err != nil {
			return fmt.Errorf("register mapping: %w", err)
		}
	}

	return nil
}
