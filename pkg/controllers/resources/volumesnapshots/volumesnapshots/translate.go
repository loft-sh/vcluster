package volumesnapshots

import (
	"context"
	"fmt"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *syncer) translate(ctx context.Context, vVS *volumesnapshotv1.VolumeSnapshot) (*volumesnapshotv1.VolumeSnapshot, error) {
	target, err := s.translator.Translate(vVS)
	if err != nil {
		return nil, err
	}
	pVS := target.(*volumesnapshotv1.VolumeSnapshot)

	if vVS.Annotations != nil && vVS.Annotations[constants.SkipTranslationAnnotation] == "true" {
		pVS.Spec.Source = vVS.Spec.Source
	} else {
		if vVS.Spec.Source.PersistentVolumeClaimName != nil {
			pvcName := translate.PhysicalName(*vVS.Spec.Source.PersistentVolumeClaimName, vVS.Namespace)
			pVS.Spec.Source.PersistentVolumeClaimName = &pvcName
		}
		if vVS.Spec.Source.VolumeSnapshotContentName != nil {
			vVSC := &volumesnapshotv1.VolumeSnapshotContent{}
			err := s.virtualClient.Get(ctx, client.ObjectKey{Name: *vVS.Spec.Source.VolumeSnapshotContentName}, vVSC)
			if err != nil {
				return nil, fmt.Errorf("failed to get virtual VolumeSnapshotContent resource referenced as source of the %s VolumeSnapshot: %v", vVS.Name, err)
			}
			translatedName := s.volumeSnapshotContentNameTranslator(vVSC.Name, vVSC)
			pVS.Spec.Source.VolumeSnapshotContentName = &translatedName
		}
	}

	pVS.Spec.VolumeSnapshotClassName = vVS.Spec.VolumeSnapshotClassName
	return pVS, nil
}

func (s *syncer) translateUpdate(pVS, vVS *volumesnapshotv1.VolumeSnapshot) *volumesnapshotv1.VolumeSnapshot {
	var updated *volumesnapshotv1.VolumeSnapshot

	// snapshot class can be updated
	if pVS.Spec.VolumeSnapshotClassName != vVS.Spec.VolumeSnapshotClassName {
		updated = newIfNil(updated, pVS)
		updated.Spec.VolumeSnapshotClassName = vVS.Spec.VolumeSnapshotClassName
	}

	updatedAnnotations := s.translator.TranslateAnnotations(vVS, pVS)
	if !equality.Semantic.DeepEqual(updatedAnnotations, pVS.Annotations) {
		updated = newIfNil(updated, pVS)
		updated.Annotations = updatedAnnotations
	}

	updatedLabels := s.translator.TranslateLabels(vVS)
	if !equality.Semantic.DeepEqual(updatedLabels, pVS.Labels) {
		updated = newIfNil(updated, pVS)
		updated.Labels = updatedLabels
	}

	return updated
}

func (s *syncer) translateUpdateBackwards(pObj, vObj *volumesnapshotv1.VolumeSnapshot) *volumesnapshotv1.VolumeSnapshot {
	var updated *volumesnapshotv1.VolumeSnapshot

	// sync back the finalizers
	if !equality.Semantic.DeepEqual(vObj.Finalizers, pObj.Finalizers) {
		updated = newIfNil(updated, vObj)
		updated.Finalizers = pObj.Finalizers
	}
	return updated
}

func newIfNil(updated *volumesnapshotv1.VolumeSnapshot, objBase *volumesnapshotv1.VolumeSnapshot) *volumesnapshotv1.VolumeSnapshot {
	if updated == nil {
		return objBase.DeepCopy()
	}
	return updated
}
