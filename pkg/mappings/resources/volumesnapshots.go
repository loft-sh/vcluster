package resources

import (
	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/util/translate"
)

func RegisterVolumeSnapshotsMapper(ctx *synccontext.RegisterContext) error {
	var (
		mapper mappings.Mapper
		err    error
	)
	if !ctx.Config.Sync.ToHost.VolumeSnapshots.Enabled {
		mapper, err = generic.NewMirrorPhysicalMapper(&volumesnapshotv1.VolumeSnapshot{})
	} else {
		mapper, err = generic.NewNamespacedMapper(ctx, &volumesnapshotv1.VolumeSnapshot{}, translate.Default.PhysicalName)
	}
	if err != nil {
		return err
	}

	return mappings.Default.AddMapper(mapper)
}
