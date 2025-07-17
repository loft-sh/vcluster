package volume

import (
	"context"

	snapshotsv1api "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
)

type Restorer interface {
	RestoreVolumes(ctx context.Context, volumeSnapshots []snapshotsv1api.VolumeSnapshot) error
}
