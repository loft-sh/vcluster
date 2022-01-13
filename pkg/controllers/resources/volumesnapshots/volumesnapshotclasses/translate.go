package volumesnapshotclasses

import (
	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (s *volumeSnapshotClassSyncer) translateBackwards(pVSC *volumesnapshotv1.VolumeSnapshotClass) *volumesnapshotv1.VolumeSnapshotClass {
	// build virtual persistent volume
	vObj := pVSC.DeepCopy()
	vObj.ResourceVersion = ""
	vObj.UID = ""
	vObj.ManagedFields = nil
	return vObj
}

func (s *volumeSnapshotClassSyncer) translateUpdateBackwards(pVSC *volumesnapshotv1.VolumeSnapshotClass, vVSC *volumesnapshotv1.VolumeSnapshotClass) *volumesnapshotv1.VolumeSnapshotClass {
	var updated *volumesnapshotv1.VolumeSnapshotClass

	if !equality.Semantic.DeepEqual(vVSC.Driver, pVSC.Driver) {
		updated = newIfNil(updated, vVSC)
		updated.Driver = pVSC.Driver
	}

	if !equality.Semantic.DeepEqual(vVSC.Parameters, pVSC.Parameters) {
		updated = newIfNil(updated, vVSC)
		updated.Parameters = pVSC.Parameters
	}

	if !equality.Semantic.DeepEqual(vVSC.DeletionPolicy, pVSC.DeletionPolicy) {
		updated = newIfNil(updated, vVSC)
		updated.DeletionPolicy = pVSC.DeletionPolicy
	}

	if !equality.Semantic.DeepEqual(vVSC.Annotations, pVSC.Annotations) {
		updated = newIfNil(updated, vVSC)
		updated.Annotations = pVSC.Annotations
	}

	if !equality.Semantic.DeepEqual(vVSC.Labels, pVSC.Labels) {
		updated = newIfNil(updated, vVSC)
		updated.Labels = pVSC.Labels
	}

	return updated
}

func newIfNil(updated *volumesnapshotv1.VolumeSnapshotClass, objBase *volumesnapshotv1.VolumeSnapshotClass) *volumesnapshotv1.VolumeSnapshotClass {
	if updated == nil {
		return objBase.DeepCopy()
	}
	return updated
}
