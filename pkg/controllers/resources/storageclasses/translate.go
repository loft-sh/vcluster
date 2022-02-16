package storageclasses

import (
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (s *storageClassSyncer) translate(vStorageClass *storagev1.StorageClass) *storagev1.StorageClass {
	return s.TranslateMetadata(vStorageClass).(*storagev1.StorageClass)
}

func (s *storageClassSyncer) translateUpdate(pObj, vObj *storagev1.StorageClass) *storagev1.StorageClass {
	var updated *storagev1.StorageClass

	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(vObj, pObj)
	if changed {
		updated = newIfNil(updated, pObj)
		updated.Labels = updatedLabels
		updated.Annotations = updatedAnnotations
	}

	if !equality.Semantic.DeepEqual(vObj.Provisioner, pObj.Provisioner) {
		updated = newIfNil(updated, pObj)
		updated.Provisioner = vObj.Provisioner
	}

	if !equality.Semantic.DeepEqual(vObj.Parameters, pObj.Parameters) {
		updated = newIfNil(updated, pObj)
		updated.Parameters = vObj.Parameters
	}

	if !equality.Semantic.DeepEqual(vObj.ReclaimPolicy, pObj.ReclaimPolicy) {
		updated = newIfNil(updated, pObj)
		updated.ReclaimPolicy = vObj.ReclaimPolicy
	}

	if !equality.Semantic.DeepEqual(vObj.MountOptions, pObj.MountOptions) {
		updated = newIfNil(updated, pObj)
		updated.MountOptions = vObj.MountOptions
	}

	if !equality.Semantic.DeepEqual(vObj.AllowVolumeExpansion, pObj.AllowVolumeExpansion) {
		updated = newIfNil(updated, pObj)
		updated.AllowVolumeExpansion = vObj.AllowVolumeExpansion
	}

	if !equality.Semantic.DeepEqual(vObj.VolumeBindingMode, pObj.VolumeBindingMode) {
		updated = newIfNil(updated, pObj)
		updated.VolumeBindingMode = vObj.VolumeBindingMode
	}

	if !equality.Semantic.DeepEqual(vObj.AllowedTopologies, pObj.AllowedTopologies) {
		updated = newIfNil(updated, pObj)
		updated.AllowedTopologies = vObj.AllowedTopologies
	}

	return updated
}

func newIfNil(updated *storagev1.StorageClass, obj *storagev1.StorageClass) *storagev1.StorageClass {
	if updated == nil {
		return obj.DeepCopy()
	}
	return updated
}
