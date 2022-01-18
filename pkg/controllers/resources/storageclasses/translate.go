package storageclasses

import (
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (s *storageClassSyncer) translate(pStorageClass *storagev1.StorageClass) *storagev1.StorageClass {
	vObj := pStorageClass.DeepCopy()
	vObj.ResourceVersion = ""
	vObj.UID = ""
	vObj.ManagedFields = nil
	return vObj
}

func (s *storageClassSyncer) translateUpdate(pObj, vObj *storagev1.StorageClass) *storagev1.StorageClass {
	var updated *storagev1.StorageClass

	if !equality.Semantic.DeepEqual(vObj.ObjectMeta.Labels, pObj.ObjectMeta.Labels) {
		updated = newIfNil(updated, vObj)
		updated.Labels = pObj.Labels
	}

	if !equality.Semantic.DeepEqual(vObj.ObjectMeta.Annotations, pObj.ObjectMeta.Annotations) {
		updated = newIfNil(updated, vObj)
		updated.Annotations = pObj.Annotations
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

func newIfNil(updated *storagev1.StorageClass, vObj *storagev1.StorageClass) *storagev1.StorageClass {
	if updated == nil {
		return vObj.DeepCopy()
	}
	return updated
}
