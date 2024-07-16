package csinodes

import (
	"context"

	storagev1 "k8s.io/api/storage/v1"
)

func (s *csinodeSyncer) translateBackwards(ctx context.Context, pCSINode *storagev1.CSINode) *storagev1.CSINode {
	return s.TranslateMetadata(ctx, pCSINode).(*storagev1.CSINode)
}

func (s *csinodeSyncer) translateUpdateBackwards(ctx context.Context, pObj, vObj *storagev1.CSINode) {
	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	if changed {
		vObj.Labels = updatedLabels
		vObj.Annotations = updatedAnnotations
	}

	pObj.Spec.DeepCopyInto(&vObj.Spec)
}
