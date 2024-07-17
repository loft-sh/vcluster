package resources

import (
	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/mappings"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
)

func CreateVolumeSnapshotClassesMapper(_ *synccontext.RegisterContext) (mappings.Mapper, error) {
	return generic.NewMirrorMapper(&volumesnapshotv1.VolumeSnapshotClass{})
}
