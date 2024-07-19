package resources

import (
	_ "embed"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/util"
)

//go:embed volumesnapshotclasses.crd.yaml
var volumeSnapshotClassesCRD string

func CreateVolumeSnapshotClassesMapper(ctx *synccontext.RegisterContext) (mappings.Mapper, error) {
	if !ctx.Config.Sync.ToHost.VolumeSnapshots.Enabled {
		return generic.NewMirrorMapper(&volumesnapshotv1.VolumeSnapshotClass{})
	}

	err := util.EnsureCRD(ctx.Context, ctx.VirtualManager.GetConfig(), []byte(volumeSnapshotClassesCRD), volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotClass"))
	if err != nil {
		return nil, err
	}

	return generic.NewMirrorMapper(&volumesnapshotv1.VolumeSnapshotClass{})
}
