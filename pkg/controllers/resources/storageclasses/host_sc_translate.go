package storageclasses

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (s *hostStorageClassSyncer) translateBackwards(ctx context.Context, pStorageClass *storagev1.StorageClass) *storagev1.StorageClass {
	return s.TranslateMetadata(ctx, pStorageClass).(*storagev1.StorageClass)
}

func (s *hostStorageClassSyncer) translateUpdateBackwards(ctx context.Context, pObj, vObj *storagev1.StorageClass) *storagev1.StorageClass {
	var updated *storagev1.StorageClass

	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	if changed {
		updated = translator.NewIfNil(updated, vObj)
		updated.Labels = updatedLabels
		updated.Annotations = updatedAnnotations
	}

	if !equality.Semantic.DeepEqual(vObj.Provisioner, pObj.Provisioner) {
		updated = translator.NewIfNil(updated, vObj)
		updated.Provisioner = pObj.Provisioner
	}

	if !equality.Semantic.DeepEqual(vObj.Parameters, pObj.Parameters) {
		updated = translator.NewIfNil(updated, vObj)
		updated.Parameters = pObj.Parameters
	}

	if !equality.Semantic.DeepEqual(vObj.ReclaimPolicy, pObj.ReclaimPolicy) {
		updated = translator.NewIfNil(updated, vObj)
		updated.ReclaimPolicy = pObj.ReclaimPolicy
	}

	if !equality.Semantic.DeepEqual(vObj.MountOptions, pObj.MountOptions) {
		updated = translator.NewIfNil(updated, vObj)
		updated.MountOptions = pObj.MountOptions
	}

	if !equality.Semantic.DeepEqual(vObj.AllowVolumeExpansion, pObj.AllowVolumeExpansion) {
		updated = translator.NewIfNil(updated, vObj)
		updated.AllowVolumeExpansion = pObj.AllowVolumeExpansion
	}

	if !equality.Semantic.DeepEqual(vObj.VolumeBindingMode, pObj.VolumeBindingMode) {
		updated = translator.NewIfNil(updated, vObj)
		updated.VolumeBindingMode = pObj.VolumeBindingMode
	}

	if !equality.Semantic.DeepEqual(vObj.AllowedTopologies, pObj.AllowedTopologies) {
		updated = translator.NewIfNil(updated, vObj)
		updated.AllowedTopologies = pObj.AllowedTopologies
	}

	return updated
}
