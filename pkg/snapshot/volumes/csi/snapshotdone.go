package csi

import (
	"context"
	"fmt"

	"github.com/loft-sh/api/v4/pkg/snapshot"
)

func (s *VolumeSnapshotter) reconcileDone(_ context.Context, requestName string, status *snapshot.VolumeSnapshotsStatus) error {
	if !status.Done() {
		return fmt.Errorf(
			"invalid phase for snapshot request %s, expected %s, %s, %s, %s or %s, got %s",
			requestName,
			snapshot.VolumeSnapshotPhaseCompleted,
			snapshot.VolumeSnapshotPhasePartiallyFailed,
			snapshot.VolumeSnapshotPhaseFailed,
			snapshot.VolumeSnapshotPhaseSkipped,
			snapshot.VolumeSnapshotPhaseCanceled,
			status.Phase)
	}
	s.logger.Debugf("Finished reconciling volume snapshots request %s, final status is %s", requestName, status.Phase)
	return nil
}
