package storageclasses

import (
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (s *hostStorageClassSyncer) translateBackwards(pStorageClass *storagev1.StorageClass) *storagev1.StorageClass {
	return s.TranslateMetadata(pStorageClass).(*storagev1.StorageClass)
}

func (s *hostStorageClassSyncer) translateUpdateBackwards(pObj, vObj *storagev1.StorageClass) *storagev1.StorageClass {
	var updated *storagev1.StorageClass

	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(vObj, pObj)
	if changed {
		updated = newIfNil(updated, vObj)
		updated.Labels = updatedLabels
		updated.Annotations = updatedAnnotations
	}

	if !equality.Semantic.DeepEqual(vObj.Provisioner, pObj.Provisioner) {
		updated = newIfNil(updated, vObj)
		updated.Provisioner = pObj.Provisioner
	}

	if !equality.Semantic.DeepEqual(vObj.Parameters, pObj.Parameters) {
		updated = newIfNil(updated, vObj)
		updated.Parameters = pObj.Parameters
	}

	if !equality.Semantic.DeepEqual(vObj.ReclaimPolicy, pObj.ReclaimPolicy) {
		updated = newIfNil(updated, vObj)
		updated.ReclaimPolicy = pObj.ReclaimPolicy
	}

	if !equality.Semantic.DeepEqual(vObj.MountOptions, pObj.MountOptions) {
		updated = newIfNil(updated, vObj)
		updated.MountOptions = pObj.MountOptions
	}

	if !equality.Semantic.DeepEqual(vObj.AllowVolumeExpansion, pObj.AllowVolumeExpansion) {
		updated = newIfNil(updated, vObj)
		updated.AllowVolumeExpansion = pObj.AllowVolumeExpansion
	}

	if !equality.Semantic.DeepEqual(vObj.VolumeBindingMode, pObj.VolumeBindingMode) {
		updated = newIfNil(updated, vObj)
		updated.VolumeBindingMode = pObj.VolumeBindingMode
	}

	if !equality.Semantic.DeepEqual(vObj.AllowedTopologies, pObj.AllowedTopologies) {
		updated = newIfNil(updated, vObj)
		updated.AllowedTopologies = pObj.AllowedTopologies
	}

	return updated
}
