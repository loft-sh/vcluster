package csistoragecapacities

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/constants"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	syncer "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ syncer.Syncer = &csistoragecapacitySyncer{}

func (s *csistoragecapacitySyncer) Name() string {
	return "csistoragecapacity"
}

func (s *csistoragecapacitySyncer) Resource() client.Object {
	return &storagev1.CSIStorageCapacity{}
}

func (s *csistoragecapacitySyncer) IsManaged(context.Context, client.Object) (bool, error) {
	return true, nil
}

func (s *csistoragecapacitySyncer) RegisterIndices(ctx *synccontext.RegisterContext) error {
	return ctx.PhysicalManager.GetFieldIndexer().IndexField(ctx.Context, &storagev1.CSIStorageCapacity{}, constants.IndexByVirtualName, func(rawObj client.Object) []string {
		return []string{s.HostToVirtual(ctx.Context, types.NamespacedName{Name: rawObj.GetName(), Namespace: rawObj.GetNamespace()}, rawObj).Name}
	})
}

// translate namespace
func (s *csistoragecapacitySyncer) HostToVirtual(_ context.Context, req types.NamespacedName, _ client.Object) types.NamespacedName {
	return types.NamespacedName{Name: translate.SafeConcatName(req.Name, "x", req.Namespace), Namespace: "kube-system"}
}
func (s *csistoragecapacitySyncer) VirtualToHost(ctx context.Context, req types.NamespacedName, vObj client.Object) types.NamespacedName {
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

// TranslateMetadata translates the object's metadata
func (s *csistoragecapacitySyncer) TranslateMetadata(ctx context.Context, pObj client.Object) (client.Object, error) {
	name := s.HostToVirtual(ctx, types.NamespacedName{Name: pObj.GetName(), Namespace: pObj.GetNamespace()}, pObj)
	pObjCopy := pObj.DeepCopyObject()
	vObj, ok := pObjCopy.(client.Object)
	if !ok {
		return nil, fmt.Errorf("%q not a metadata object: %+v", pObj.GetName(), pObjCopy)
	}
	translate.ResetObjectMetadata(vObj)
	vObj.SetName(name.Name)
	vObj.SetNamespace(name.Namespace)
	vObj.SetAnnotations(translate.Default.ApplyAnnotations(pObj, nil, []string{}))
	vObj.SetLabels(translate.Default.ApplyLabels(pObj, nil, []string{}))
	return vObj, nil
}

// TranslateMetadataUpdate translates the object's metadata annotations and labels and determines
// if they have changed between the physical and virtual object
func (s *csistoragecapacitySyncer) TranslateMetadataUpdate(vObj client.Object, pObj client.Object) (changed bool, annotations map[string]string, labels map[string]string) {
	updatedAnnotations := translate.Default.ApplyAnnotations(pObj, vObj, []string{})
	updatedLabels := translate.Default.ApplyLabels(pObj, vObj, []string{})
	return !equality.Semantic.DeepEqual(updatedAnnotations, vObj.GetAnnotations()) || !equality.Semantic.DeepEqual(updatedLabels, vObj.GetLabels()), updatedAnnotations, updatedLabels
}
