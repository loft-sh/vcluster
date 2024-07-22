package csistoragecapacities

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncer "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ syncer.Syncer = &csistoragecapacitySyncer{}

func (s *csistoragecapacitySyncer) Name() string {
	return "csistoragecapacity"
}

func (s *csistoragecapacitySyncer) Resource() client.Object {
	return &storagev1.CSIStorageCapacity{}
}

// TranslateMetadata translates the object's metadata
func (s *csistoragecapacitySyncer) TranslateMetadata(ctx *synccontext.SyncContext, pObj client.Object) (client.Object, error) {
	pName := mappings.HostToVirtual(ctx, pObj.GetName(), pObj.GetNamespace(), pObj, mappings.CSIStorageCapacities())
	pObjCopy := pObj.DeepCopyObject()
	vObj, ok := pObjCopy.(client.Object)
	if !ok {
		return nil, fmt.Errorf("%q not a metadata object: %+v", pObj.GetName(), pObjCopy)
	}
	translate.ResetObjectMetadata(vObj)
	vObj.SetName(pName.Name)
	vObj.SetNamespace(pName.Namespace)
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
