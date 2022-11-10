package csidrivers

import (
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (s *csidriverSyncer) translateBackwards(pCSIDriver *storagev1.CSIDriver) *storagev1.CSIDriver {
	return s.TranslateMetadata(pCSIDriver).(*storagev1.CSIDriver)
}

func (s *csidriverSyncer) translateUpdateBackwards(pObj, vObj *storagev1.CSIDriver) *storagev1.CSIDriver {
	var updated *storagev1.CSIDriver

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

func newIfNil(updated *storagev1.CSIDriver, obj *storagev1.CSIDriver) *storagev1.CSIDriver {
	if updated == nil {
		return obj.DeepCopy()
	}
	return updated
}
