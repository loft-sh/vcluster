package csinodes

import (
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
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
		updated = translator.NewIfNil(updated, vObj)
		updated.Labels = updatedLabels
		updated.Annotations = updatedAnnotations
	}

	if !equality.Semantic.DeepEqual(vObj.Spec, pObj.Spec) {
		updated = translator.NewIfNil(updated, vObj)
		pObj.Spec.DeepCopyInto(&updated.Spec)
	}

	return updated
}
