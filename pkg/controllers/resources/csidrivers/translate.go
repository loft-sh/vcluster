package csidrivers

import (
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	storagev1 "k8s.io/api/storage/v1"
)

func (s *csidriverSyncer) translateBackwards(ctx *synccontext.SyncContext, pCSIDriver *storagev1.CSIDriver) *storagev1.CSIDriver {
	return s.TranslateMetadata(ctx, pCSIDriver).(*storagev1.CSIDriver)
}

func (s *csidriverSyncer) translateUpdateBackwards(ctx *synccontext.SyncContext, pObj, vObj *storagev1.CSIDriver) {
	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	if changed {
		vObj.Labels = updatedLabels
		vObj.Annotations = updatedAnnotations
	}

	pObj.Spec.DeepCopyInto(&vObj.Spec)
}
