package csi

import (
	"context"
	"fmt"

	"github.com/loft-sh/api/v4/pkg/snapshot"

	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
)

func (r *Restorer) reconcileDone(_ context.Context, requestName string, status *volumes.RestoreRequestStatus) error {
	if !status.Done() {
		return fmt.Errorf(
			"invalid phase for snapshot request %s, expected %s, %s, %s or %s, got %s",
			requestName,
			snapshot.VolumeSnapshotPhaseCompleted,
			snapshot.VolumeSnapshotPhasePartiallyFailed,
			snapshot.VolumeSnapshotPhaseFailed,
			snapshot.VolumeSnapshotPhaseSkipped,
			status.Phase)
	}
	r.logger.Debugf("Finished reconciling volumes restore request %s, final status is %s", requestName, status.Phase)
	return nil
}
