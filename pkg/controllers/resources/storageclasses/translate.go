package storageclasses

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (s *storageClassSyncer) translate(ctx context.Context, vStorageClass *storagev1.StorageClass) *storagev1.StorageClass {
	return s.TranslateMetadata(ctx, vStorageClass).(*storagev1.StorageClass)
}

func (s *storageClassSyncer) translateUpdate(ctx context.Context, pObj, vObj *storagev1.StorageClass) *storagev1.StorageClass {
	var updated *storagev1.StorageClass

	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	if changed {
		updated = translator.NewIfNil(updated, pObj)
		updated.Labels = updatedLabels
		updated.Annotations = updatedAnnotations
	}

	if !equality.Semantic.DeepEqual(vObj.Provisioner, pObj.Provisioner) {
		updated = translator.NewIfNil(updated, pObj)
		updated.Provisioner = vObj.Provisioner
	}

	if !equality.Semantic.DeepEqual(vObj.Parameters, pObj.Parameters) {
		updated = translator.NewIfNil(updated, pObj)
		updated.Parameters = vObj.Parameters
	}

	if !equality.Semantic.DeepEqual(vObj.ReclaimPolicy, pObj.ReclaimPolicy) {
		updated = translator.NewIfNil(updated, pObj)
		updated.ReclaimPolicy = vObj.ReclaimPolicy
	}

	if !equality.Semantic.DeepEqual(vObj.MountOptions, pObj.MountOptions) {
		updated = translator.NewIfNil(updated, pObj)
		updated.MountOptions = vObj.MountOptions
	}

	if !equality.Semantic.DeepEqual(vObj.AllowVolumeExpansion, pObj.AllowVolumeExpansion) {
		updated = translator.NewIfNil(updated, pObj)
		updated.AllowVolumeExpansion = vObj.AllowVolumeExpansion
	}

	if !equality.Semantic.DeepEqual(vObj.VolumeBindingMode, pObj.VolumeBindingMode) {
		updated = translator.NewIfNil(updated, pObj)
		updated.VolumeBindingMode = vObj.VolumeBindingMode
	}

	if !equality.Semantic.DeepEqual(vObj.AllowedTopologies, pObj.AllowedTopologies) {
		updated = translator.NewIfNil(updated, pObj)
		updated.AllowedTopologies = vObj.AllowedTopologies
	}

	return updated
}
