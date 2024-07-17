package resources

import (
	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/util/translate"
)

func CreateVolumeSnapshotsMapper(ctx *synccontext.RegisterContext) (mappings.Mapper, error) {
	if !ctx.Config.Sync.ToHost.VolumeSnapshots.Enabled {
		return generic.NewMirrorMapper(&volumesnapshotv1.VolumeSnapshot{})
	}

	return generic.NewMapper(ctx, &volumesnapshotv1.VolumeSnapshot{}, translate.Default.PhysicalName)
}
