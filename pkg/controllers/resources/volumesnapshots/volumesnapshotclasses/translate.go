package volumesnapshotclasses

import (
	"context"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (s *volumeSnapshotClassSyncer) translateBackwards(ctx context.Context, pVSC *volumesnapshotv1.VolumeSnapshotClass) *volumesnapshotv1.VolumeSnapshotClass {
	return s.TranslateMetadata(ctx, pVSC).(*volumesnapshotv1.VolumeSnapshotClass)
}

func (s *volumeSnapshotClassSyncer) translateUpdateBackwards(ctx context.Context, pVSC *volumesnapshotv1.VolumeSnapshotClass, vVSC *volumesnapshotv1.VolumeSnapshotClass) {
	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vVSC, pVSC)
	if changed {
		vVSC.Labels = updatedLabels
		vVSC.Annotations = updatedAnnotations
	}

	if !equality.Semantic.DeepEqual(vVSC.Driver, pVSC.Driver) {
		vVSC.Driver = pVSC.Driver
	}

	if !equality.Semantic.DeepEqual(vVSC.Parameters, pVSC.Parameters) {
		vVSC.Parameters = pVSC.Parameters
	}

	if !equality.Semantic.DeepEqual(vVSC.DeletionPolicy, pVSC.DeletionPolicy) {
		vVSC.DeletionPolicy = pVSC.DeletionPolicy
	}
}
