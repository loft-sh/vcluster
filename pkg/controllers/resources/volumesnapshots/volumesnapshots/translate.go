package volumesnapshots

import (
	"fmt"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *volumeSnapshotSyncer) translate(ctx *synccontext.SyncContext, vVS *volumesnapshotv1.VolumeSnapshot) (*volumesnapshotv1.VolumeSnapshot, error) {
	pVS := translate.HostMetadata(vVS, s.VirtualToHost(ctx, types.NamespacedName{Name: vVS.GetName(), Namespace: vVS.GetNamespace()}, vVS))
	if vVS.Annotations != nil && vVS.Annotations[constants.SkipTranslationAnnotation] == "true" {
		pVS.Spec.Source = vVS.Spec.Source
	} else {
		if vVS.Spec.Source.PersistentVolumeClaimName != nil {
			pvcName := mappings.VirtualToHostName(ctx, *vVS.Spec.Source.PersistentVolumeClaimName, vVS.Namespace, mappings.PersistentVolumeClaims())
			pVS.Spec.Source.PersistentVolumeClaimName = &pvcName
		}
		if vVS.Spec.Source.VolumeSnapshotContentName != nil {
			vVSC := &volumesnapshotv1.VolumeSnapshotContent{}
			err := ctx.VirtualClient.Get(ctx, client.ObjectKey{Name: *vVS.Spec.Source.VolumeSnapshotContentName}, vVSC)
			if err != nil {
				return nil, fmt.Errorf("failed to get virtual VolumeSnapshotContent resource referenced as source of the %s VolumeSnapshot: %w", vVS.Name, err)
			}

			mapper, err := ctx.Mappings.ByGVK(mappings.VolumeSnapshotContents())
			if err != nil {
				return nil, err
			}

			translatedName := mapper.VirtualToHost(ctx, types.NamespacedName{Name: vVSC.Name}, vVSC).Name
			pVS.Spec.Source.VolumeSnapshotContentName = &translatedName
		}
	}

	pVS.Spec.VolumeSnapshotClassName = vVS.Spec.VolumeSnapshotClassName
	return pVS, nil
}
