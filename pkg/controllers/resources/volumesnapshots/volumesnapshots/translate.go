package volumesnapshots

import (
	"context"
	"fmt"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"github.com/loft-sh/vcluster/pkg/constants"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *volumeSnapshotSyncer) translate(ctx *synccontext.SyncContext, vVS *volumesnapshotv1.VolumeSnapshot) (*volumesnapshotv1.VolumeSnapshot, error) {
	pVS := s.TranslateMetadata(ctx.Context, vVS).(*volumesnapshotv1.VolumeSnapshot)
	if vVS.Annotations != nil && vVS.Annotations[constants.SkipTranslationAnnotation] == "true" {
		pVS.Spec.Source = vVS.Spec.Source
	} else {
		if vVS.Spec.Source.PersistentVolumeClaimName != nil {
			pvcName := translate.Default.PhysicalName(*vVS.Spec.Source.PersistentVolumeClaimName, vVS.Namespace)
			pVS.Spec.Source.PersistentVolumeClaimName = &pvcName
		}
		if vVS.Spec.Source.VolumeSnapshotContentName != nil {
			vVSC := &volumesnapshotv1.VolumeSnapshotContent{}
			err := ctx.VirtualClient.Get(ctx.Context, client.ObjectKey{Name: *vVS.Spec.Source.VolumeSnapshotContentName}, vVSC)
			if err != nil {
				return nil, fmt.Errorf("failed to get virtual VolumeSnapshotContent resource referenced as source of the %s VolumeSnapshot: %w", vVS.Name, err)
			}
			translatedName := s.volumeSnapshotContentNameTranslator(vVSC.Name, vVSC)
			pVS.Spec.Source.VolumeSnapshotContentName = &translatedName
		}
	}

	pVS.Spec.VolumeSnapshotClassName = vVS.Spec.VolumeSnapshotClassName
	return pVS, nil
}

func (s *volumeSnapshotSyncer) translateUpdate(ctx context.Context, pVS, vVS *volumesnapshotv1.VolumeSnapshot) *volumesnapshotv1.VolumeSnapshot {
	var updated *volumesnapshotv1.VolumeSnapshot

	// snapshot class can be updated
	if !equality.Semantic.DeepEqual(pVS.Spec.VolumeSnapshotClassName, vVS.Spec.VolumeSnapshotClassName) {
		updated = translator.NewIfNil(updated, pVS)
		updated.Spec.VolumeSnapshotClassName = vVS.Spec.VolumeSnapshotClassName
	}

	// check if metadata changed
	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vVS, pVS)
	if changed {
		updated = translator.NewIfNil(updated, pVS)
		updated.Annotations = updatedAnnotations
		updated.Labels = updatedLabels
	}

	return updated
}

func (s *volumeSnapshotSyncer) translateUpdateBackwards(pObj, vObj *volumesnapshotv1.VolumeSnapshot) *volumesnapshotv1.VolumeSnapshot {
	var updated *volumesnapshotv1.VolumeSnapshot

	// sync back the finalizers
	if !equality.Semantic.DeepEqual(vObj.Finalizers, pObj.Finalizers) {
		updated = translator.NewIfNil(updated, vObj)
		updated.Finalizers = pObj.Finalizers
	}
	return updated
}
