package csistoragecapacities

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/constants"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (s *csistoragecapacitySyncer) translateBackwards(ctx *synccontext.SyncContext, pCSIStorageCapacity *storagev1.CSIStorageCapacity) (*storagev1.CSIStorageCapacity, error) {
	translated, err := s.TranslateMetadata(pCSIStorageCapacity)
	if err != nil {
		return nil, fmt.Errorf("failed to translate metatdata backwards: %w", err)
	}
	vObj, ok := translated.(*storagev1.CSIStorageCapacity)
	if !ok {
		return nil, fmt.Errorf("failed to translate metatdata backwards: translated not a CSIStorageCapacity object: %+v", translated)
	}
	return vObj, nil
}

func (s *csistoragecapacitySyncer) translateUpdateBackwards(ctx *synccontext.SyncContext, pObj, vObj *storagev1.CSIStorageCapacity) (*storagev1.CSIStorageCapacity, error) {
	var updated *storagev1.CSIStorageCapacity

	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(vObj, pObj)
	if changed {
		updated = newIfNil(updated, vObj)
		updated.Labels = updatedLabels
		updated.Annotations = updatedAnnotations
	}

	scName, err := s.translateStorageClassNameBackwards(ctx, pObj.StorageClassName)
	if err != nil {
		return nil, err
	}

	if scName != pObj.StorageClassName {
		updated = newIfNil(updated, vObj)
		updated.StorageClassName = scName

	}

	if !equality.Semantic.DeepEqual(vObj.NodeTopology, pObj.NodeTopology) {
		updated = newIfNil(updated, vObj)
		updated.NodeTopology = pObj.NodeTopology
	}

	if !equality.Semantic.DeepEqual(vObj.Capacity, pObj.Capacity) {
		updated = newIfNil(updated, vObj)
		updated.Capacity = pObj.Capacity
	}

	if !equality.Semantic.DeepEqual(vObj.MaximumVolumeSize, pObj.MaximumVolumeSize) {
		updated = newIfNil(updated, vObj)
		updated.MaximumVolumeSize = pObj.MaximumVolumeSize
	}

	return updated, nil
}

// the storageClassName should map to the name of storageClasses present in the virtualCluster,
// so the behaviour changes according to which storageclassSyncer is enabled
func (s *csistoragecapacitySyncer) translateStorageClassNameBackwards(ctx *synccontext.SyncContext, name string) (string, error) {
	if !s.storageClassSyncEnabled {
		return name, nil
	}
	// the csistorage capacity being synced to the virtual cluster needs the name of the virtual storage cluster
	sc := &storagev1.StorageClass{}
	err := clienthelper.GetByIndex(ctx.Context, ctx.VirtualClient, sc, constants.IndexByPhysicalName, name)
	return sc.Name, err
}

func newIfNil(updated *storagev1.CSIStorageCapacity, obj *storagev1.CSIStorageCapacity) *storagev1.CSIStorageCapacity {
	if updated == nil {
		return obj.DeepCopy()
	}
	return updated
}
