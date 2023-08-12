package csidrivers

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (s *csidriverSyncer) translateBackwards(ctx context.Context, pCSIDriver *storagev1.CSIDriver) *storagev1.CSIDriver {
	return s.TranslateMetadata(ctx, pCSIDriver).(*storagev1.CSIDriver)
}

func (s *csidriverSyncer) translateUpdateBackwards(ctx context.Context, pObj, vObj *storagev1.CSIDriver) *storagev1.CSIDriver {
	var updated *storagev1.CSIDriver

	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
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
