package csidrivers

import (
	"context"

	storagev1 "k8s.io/api/storage/v1"
)

func (s *csidriverSyncer) translateBackwards(ctx context.Context, pCSIDriver *storagev1.CSIDriver) *storagev1.CSIDriver {
	return s.TranslateMetadata(ctx, pCSIDriver).(*storagev1.CSIDriver)
}

func (s *csidriverSyncer) translateUpdateBackwards(ctx context.Context, pObj, vObj *storagev1.CSIDriver) {
	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	if changed {
		vObj.Labels = updatedLabels
		vObj.Annotations = updatedAnnotations
	}

	pObj.Spec.DeepCopyInto(&vObj.Spec)
}
