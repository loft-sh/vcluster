package storageclasses

import (
	"context"

	storagev1 "k8s.io/api/storage/v1"
)

func (s *hostStorageClassSyncer) translateBackwards(ctx context.Context, pStorageClass *storagev1.StorageClass) *storagev1.StorageClass {
	return s.TranslateMetadata(ctx, pStorageClass).(*storagev1.StorageClass)
}

func (s *hostStorageClassSyncer) translateUpdateBackwards(ctx context.Context, pObj, vObj *storagev1.StorageClass) {
	_, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	vObj.Labels = updatedLabels
	vObj.Annotations = updatedAnnotations

	vObj.Provisioner = pObj.Provisioner

	vObj.Parameters = pObj.Parameters

	vObj.ReclaimPolicy = pObj.ReclaimPolicy

	vObj.MountOptions = pObj.MountOptions

	vObj.AllowVolumeExpansion = pObj.AllowVolumeExpansion

	vObj.VolumeBindingMode = pObj.VolumeBindingMode

	vObj.AllowedTopologies = pObj.AllowedTopologies
}
