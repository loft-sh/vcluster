package volumesnapshotclasses

import (
	"context"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (s *volumeSnapshotClassSyncer) translateBackwards(ctx context.Context, pVSC *volumesnapshotv1.VolumeSnapshotClass) *volumesnapshotv1.VolumeSnapshotClass {
	return s.TranslateMetadata(ctx, pVSC).(*volumesnapshotv1.VolumeSnapshotClass)
}

func (s *volumeSnapshotClassSyncer) translateUpdateBackwards(ctx context.Context, pVSC *volumesnapshotv1.VolumeSnapshotClass, vVSC *volumesnapshotv1.VolumeSnapshotClass) *volumesnapshotv1.VolumeSnapshotClass {
	var updated *volumesnapshotv1.VolumeSnapshotClass

	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vVSC, pVSC)
	if changed {
		updated = translator.NewIfNil(updated, vVSC)
		updated.Labels = updatedLabels
		updated.Annotations = updatedAnnotations
	}

	if !equality.Semantic.DeepEqual(vVSC.Driver, pVSC.Driver) {
		updated = translator.NewIfNil(updated, vVSC)
		updated.Driver = pVSC.Driver
	}

	if !equality.Semantic.DeepEqual(vVSC.Parameters, pVSC.Parameters) {
		updated = translator.NewIfNil(updated, vVSC)
		updated.Parameters = pVSC.Parameters
	}

	if !equality.Semantic.DeepEqual(vVSC.DeletionPolicy, pVSC.DeletionPolicy) {
		updated = translator.NewIfNil(updated, vVSC)
		updated.DeletionPolicy = pVSC.DeletionPolicy
	}

	return updated
}
