package csistoragecapacities

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/constants"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// returns virtual scname, shouldSync
func (s *csistoragecapacitySyncer) fetchVirtualStorageClass(ctx *synccontext.SyncContext, physName string) (string, bool, error) {
	if s.storageClassSyncEnabled {
		sc := &storagev1.StorageClass{}
		// the csistorage capacity being synced to the virtual cluster needs the name of the virtual storage cluster
		err := clienthelper.GetByIndex(ctx.Context, ctx.VirtualClient, sc, constants.IndexByPhysicalName, physName)
		if kerrors.IsNotFound(err) {
			return "", true, nil
		}
		return sc.Name, false, nil
	}
	return physName, false, nil
}

func (s *csistoragecapacitySyncer) hasMatchingVirtualNodes(ctx *synccontext.SyncContext, ls *metav1.LabelSelector) (bool, error) {
	// sync only if the capacity applies to a synced node
	if s.storageClassSyncEnabled && ls != nil {
		nodeList := &corev1.NodeList{}
		selector, err := metav1.LabelSelectorAsSelector(ls)
		if err != nil {
			return false, err
		}
		err = ctx.VirtualClient.List(ctx.Context, nodeList, client.MatchingLabelsSelector{Selector: selector})
		if err != nil {
			return false, err
		}
		return len(nodeList.Items) == 0, nil
	}
	return false, nil
}

func (s *csistoragecapacitySyncer) translateBackwards(ctx *synccontext.SyncContext, pObj *storagev1.CSIStorageCapacity) (*storagev1.CSIStorageCapacity, bool, error) {
	scName, shouldSkip, err := s.fetchVirtualStorageClass(ctx, pObj.StorageClassName)
	if shouldSkip || err != nil {
		return nil, shouldSkip, err
	}

	shouldSkip, err = s.hasMatchingVirtualNodes(ctx, pObj.NodeTopology)
	if shouldSkip || err != nil {
		return nil, shouldSkip, err
	}

	translated, err := s.TranslateMetadata(ctx.Context, pObj.DeepCopy())
	if err != nil {
		return nil, false, fmt.Errorf("failed to translate metatdata backwards: %w", err)
	}
	vObj, ok := translated.(*storagev1.CSIStorageCapacity)
	if !ok {
		return nil, false, fmt.Errorf("failed to translate metatdata backwards: translated not a CSIStorageCapacity object: %+v", translated)
	}

	vObj.StorageClassName = scName

	return vObj, false, nil
}

func (s *csistoragecapacitySyncer) translateUpdateBackwards(ctx *synccontext.SyncContext, pObj, vObj *storagev1.CSIStorageCapacity) (*storagev1.CSIStorageCapacity, bool, error) {
	var updated *storagev1.CSIStorageCapacity
	var err error

	scName, shouldSkip, err := s.fetchVirtualStorageClass(ctx, pObj.StorageClassName)
	if shouldSkip || err != nil {
		return nil, shouldSkip, err
	}

	shouldSkip, err = s.hasMatchingVirtualNodes(ctx, pObj.NodeTopology)
	if shouldSkip || err != nil {
		return nil, shouldSkip, err
	}

	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(vObj, pObj)
	if changed {
		updated = translator.NewIfNil(updated, vObj)
		updated.Labels = updatedLabels
		updated.Annotations = updatedAnnotations
	}

	if scName != vObj.StorageClassName {
		updated = translator.NewIfNil(updated, vObj)
		updated.StorageClassName = scName
	}

	if !equality.Semantic.DeepEqual(vObj.NodeTopology, pObj.NodeTopology) {
		updated = translator.NewIfNil(updated, vObj)
		updated.NodeTopology = pObj.NodeTopology
	}

	if !equality.Semantic.DeepEqual(vObj.Capacity, pObj.Capacity) {
		updated = translator.NewIfNil(updated, vObj)
		updated.Capacity = pObj.Capacity
	}

	if !equality.Semantic.DeepEqual(vObj.MaximumVolumeSize, pObj.MaximumVolumeSize) {
		updated = translator.NewIfNil(updated, vObj)
		updated.MaximumVolumeSize = pObj.MaximumVolumeSize
	}

	return updated, false, nil
}
