package csistoragecapacities

import (
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// returns virtual scname, shouldSync
func (s *csistoragecapacitySyncer) fetchVirtualStorageClass(ctx *synccontext.SyncContext, physName string) (string, bool, error) {
	if s.storageClassSyncEnabled {
		// the csistorage capacity being synced to the virtual cluster needs the name of the virtual storage cluster
		vName := mappings.HostToVirtual(ctx, physName, "", nil, mappings.StorageClasses())
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

	vObj := s.virtualMetadata(ctx, pObj.DeepCopy())
	vObj.StorageClassName = scName
	return vObj, false, nil
}

// TranslateMetadata translates the object's metadata
func (s *csistoragecapacitySyncer) virtualMetadata(ctx *synccontext.SyncContext, pObj *storagev1.CSIStorageCapacity) *storagev1.CSIStorageCapacity {
	vObj := translate.CopyObjectWithName(pObj, s.HostToVirtual(ctx, types.NamespacedName{Name: pObj.Name, Namespace: pObj.Namespace}, pObj), false)
	vObj.SetAnnotations(translate.HostAnnotations(pObj, vObj))
	vObj.SetLabels(translate.HostLabels(pObj, nil))
	return vObj
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

	vObj.Annotations = translate.HostAnnotations(pObj, vObj)
	vObj.Labels = translate.HostLabels(pObj, vObj)
	vObj.StorageClassName = scName
	vObj.NodeTopology = pObj.NodeTopology
	vObj.Capacity = pObj.Capacity
	vObj.MaximumVolumeSize = pObj.MaximumVolumeSize
	return false, nil
}
