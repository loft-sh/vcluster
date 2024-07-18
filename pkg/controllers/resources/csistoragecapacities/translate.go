package csistoragecapacities

import (
	"fmt"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// returns virtual scname, shouldSync
func (s *csistoragecapacitySyncer) fetchVirtualStorageClass(ctx *synccontext.SyncContext, physName string) (string, bool, error) {
	if s.storageClassSyncEnabled {
		// the csistorage capacity being synced to the virtual cluster needs the name of the virtual storage cluster
		vName := mappings.StorageClasses().HostToVirtual(ctx, types.NamespacedName{Name: physName}, nil)
		if vName.Name == "" {
			return "", true, nil
		}

		return vName.Name, false, nil
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
		err = ctx.VirtualClient.List(ctx, nodeList, client.MatchingLabelsSelector{Selector: selector})
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

	translated, err := s.TranslateMetadata(ctx, pObj.DeepCopy())
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

func (s *csistoragecapacitySyncer) translateUpdateBackwards(ctx *synccontext.SyncContext, pObj, vObj *storagev1.CSIStorageCapacity) (bool, error) {
	scName, shouldSkip, err := s.fetchVirtualStorageClass(ctx, pObj.StorageClassName)
	if shouldSkip || err != nil {
		return shouldSkip, err
	}

	shouldSkip, err = s.hasMatchingVirtualNodes(ctx, pObj.NodeTopology)
	if shouldSkip || err != nil {
		return shouldSkip, err
	}

	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(vObj, pObj)
	if changed {
		vObj.Labels = updatedLabels
		vObj.Annotations = updatedAnnotations
	}

	if scName != vObj.StorageClassName {
		vObj.StorageClassName = scName
	}

	if !equality.Semantic.DeepEqual(vObj.NodeTopology, pObj.NodeTopology) {
		vObj.NodeTopology = pObj.NodeTopology
	}

	if !equality.Semantic.DeepEqual(vObj.Capacity, pObj.Capacity) {
		vObj.Capacity = pObj.Capacity
	}

	if !equality.Semantic.DeepEqual(vObj.MaximumVolumeSize, pObj.MaximumVolumeSize) {
		vObj.MaximumVolumeSize = pObj.MaximumVolumeSize
	}

	return false, nil
}
