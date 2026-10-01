package resources

import (
	_ "embed"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/util"
)

//go:embed volumesnapshotclasses.crd.yaml
var volumeSnapshotClassesCRD string

func CreateVolumeSnapshotClassesMapper(ctx *synccontext.RegisterContext) (synccontext.Mapper, error) {
	if !ctx.Config.Sync.FromHost.VolumeSnapshotClasses.Enabled {
		return generic.NewMirrorMapper(&volumesnapshotv1.VolumeSnapshotClass{})
	}

	err := util.EnsureCRD(ctx.Context, ctx.VirtualManager.GetConfig(), []byte(volumeSnapshotClassesCRD), volumesnapshotv1.SchemeGroupVersion.WithKind("VolumeSnapshotClass"))
	if err != nil {
		return nil, err
	}

	return generic.NewMirrorMapper(&volumesnapshotv1.VolumeSnapshotClass{})
}
