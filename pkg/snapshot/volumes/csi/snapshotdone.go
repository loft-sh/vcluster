package csi

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
)

func (s *VolumeSnapshotter) reconcileDone(_ context.Context, requestName string, status *volumes.SnapshotsStatus) error {
	if !status.Done() {
		return fmt.Errorf(
			"invalid phase for snapshot request %s, expected %s, %s, %s, %s or %s, got %s",
			requestName,
			volumes.RequestPhaseCompleted,
			volumes.RequestPhasePartiallyFailed,
			volumes.RequestPhaseFailed,
			volumes.RequestPhaseSkipped,
			volumes.RequestPhaseCanceled,
			status.Phase)
	}
	s.logger.Debugf("Finished reconciling volume snapshots request %s, final status is %s", requestName, status.Phase)
	return nil
}
