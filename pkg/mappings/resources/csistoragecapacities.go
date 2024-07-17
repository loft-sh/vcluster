package resources

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/constants"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateCSIStorageCapacitiesMapper(ctx *synccontext.RegisterContext) (mappings.Mapper, error) {
	s := &csiStorageCapacitiesMapper{
		physicalClient: ctx.PhysicalManager.GetClient(),
	}
	err := ctx.PhysicalManager.GetFieldIndexer().IndexField(ctx.Context, &storagev1.CSIStorageCapacity{}, constants.IndexByVirtualName, func(rawObj client.Object) []string {
		return []string{s.HostToVirtual(ctx.Context, types.NamespacedName{Name: rawObj.GetName(), Namespace: rawObj.GetNamespace()}, rawObj).Name}
	})
	if err != nil {
		return nil, err
	}

	return s, nil
}

type csiStorageCapacitiesMapper struct {
	physicalClient client.Client
}

func (s *csiStorageCapacitiesMapper) GroupVersionKind() schema.GroupVersionKind {
	return storagev1.SchemeGroupVersion.WithKind("CSIStorageCapacity")
}

func (s *csiStorageCapacitiesMapper) HostToVirtual(_ context.Context, req types.NamespacedName, _ client.Object) types.NamespacedName {
	return types.NamespacedName{Name: translate.SafeConcatName(req.Name, "x", req.Namespace), Namespace: "kube-system"}
}

func (s *csiStorageCapacitiesMapper) VirtualToHost(ctx context.Context, req types.NamespacedName, vObj client.Object) types.NamespacedName {
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

	sc := &storagev1.CSIStorageCapacity{}
	pObj := sc.DeepCopyObject().(client.Object)
	err := clienthelper.GetByIndex(ctx, s.physicalClient, pObj, constants.IndexByVirtualName, req.Name)
	if err != nil {
		return types.NamespacedName{}
	}

	return types.NamespacedName{
		Namespace: pObj.GetNamespace(),
		Name:      pObj.GetName(),
	}
}
