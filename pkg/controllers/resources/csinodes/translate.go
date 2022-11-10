package csinodes

import (
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (s *csinodeSyncer) translateBackwards(pCSINode *storagev1.CSINode) *storagev1.CSINode {
	return s.TranslateMetadata(pCSINode).(*storagev1.CSINode)
}

func (s *csinodeSyncer) translateUpdateBackwards(pObj, vObj *storagev1.CSINode) *storagev1.CSINode {
	var updated *storagev1.CSINode

	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(vObj, pObj)
	if changed {
		updated = newIfNil(updated, vObj)
		updated.Labels = updatedLabels
		updated.Annotations = updatedAnnotations
	}

	if !equality.Semantic.DeepEqual(vObj.Spec, pObj.Spec) {
		updated = newIfNil(updated, vObj)
		pObj.Spec.DeepCopyInto(&updated.Spec)
	}

	return updated
}

func newIfNil(updated *storagev1.CSINode, obj *storagev1.CSINode) *storagev1.CSINode {
	if updated == nil {
		return obj.DeepCopy()
	}
	return updated
}
