package csistoragecapacities

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateCSIStorageCapacitiesMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	return generic.WithRecorder(&csiStorageCapacitiesMapper{
		physicalClient: ctx.PhysicalManager.GetClient(),
	}), nil
}

type csiStorageCapacitiesMapper struct {
	physicalClient client.Client
}

func (s *csiStorageCapacitiesMapper) Migrate(ctx *synccontext.RegisterContext, mapper synccontext.Mapper) error {
	list := &storagev1.CSIStorageCapacityList{}
	err := ctx.VirtualManager.GetClient().List(ctx, list)
	if err != nil {
		return fmt.Errorf("error listing csi storage capacities: %w", err)
	}

	for _, val := range list.Items {
		item := &val

		// this will try to translate and record the mapping
		vName := types.NamespacedName{Name: item.Name, Namespace: item.Namespace}
		pName := mapper.VirtualToHost(ctx.ToSyncContext("migrate-"+item.Kind), vName, item)
		if pName.Name != "" {
			nameMapping := synccontext.NameMapping{
				GroupVersionKind: s.GroupVersionKind(),
				VirtualName:      vName,
				HostName:         pName,
			}

			err = ctx.Mappings.Store().AddReferenceAndSave(ctx, nameMapping, nameMapping)
			if err != nil {
				return fmt.Errorf("error saving reference in store: %w", err)
			}
		}
	}

	return nil
}

func (s *csiStorageCapacitiesMapper) GroupVersionKind() schema.GroupVersionKind {
	return storagev1.SchemeGroupVersion.WithKind("CSIStorageCapacity")
}

func (s *csiStorageCapacitiesMapper) HostToVirtual(_ *synccontext.SyncContext, req types.NamespacedName, _ client.Object) types.NamespacedName {
	return types.NamespacedName{Name: translate.SafeConcatName(req.Name, "x", req.Namespace), Namespace: "kube-system"}
}

func (s *csiStorageCapacitiesMapper) VirtualToHost(_ *synccontext.SyncContext, _ types.NamespacedName, vObj client.Object) types.NamespacedName {
	// if the virtual object is annotated with the physical name and namespace, return that
	if vObj != nil {
		vAnnotations := vObj.GetAnnotations()
		if vAnnotations != nil && vAnnotations[translate.NameAnnotation] != "" {
			return types.NamespacedName{
				Namespace: vAnnotations[translate.NamespaceAnnotation],
				Name:      vAnnotations[translate.NameAnnotation],
			}
		}
	}

	return types.NamespacedName{}
}

func (s *csiStorageCapacitiesMapper) IsManaged(*synccontext.SyncContext, client.Object) (bool, error) {
	return true, nil
}
