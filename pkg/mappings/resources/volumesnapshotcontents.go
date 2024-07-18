package resources

import (
	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	"github.com/loft-sh/vcluster/pkg/constants"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateVolumeSnapshotContentsMapper(ctx *synccontext.RegisterContext) (mappings.Mapper, error) {
	if !ctx.Config.Sync.ToHost.VolumeSnapshots.Enabled {
		return generic.NewMirrorMapper(&volumesnapshotv1.VolumeSnapshotContent{})
	}

	return generic.NewMapperWithObject(ctx, &volumesnapshotv1.VolumeSnapshotContent{}, func(name, _ string, vObj client.Object) string {
		if vObj == nil {
			return name
		}

		vVSC, ok := vObj.(*volumesnapshotv1.VolumeSnapshotContent)
		if !ok || vVSC.Annotations == nil || vVSC.Annotations[constants.HostClusterVSCAnnotation] == "" {
			return translate.Default.PhysicalNameClusterScoped(name)
		}

		return vVSC.Annotations[constants.HostClusterVSCAnnotation]
	})
}
