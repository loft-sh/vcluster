package csi

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
)

func (s *VolumeSnapshotter) reconcileCompleted(_ context.Context, snapshotRequestName string, snapshotRequest *volumes.SnapshotRequest) error {
	s.logger.Debugf("Reconciling completed volume snapshots request %s", snapshotRequestName)
	if snapshotRequest.Status.Phase != volumes.RequestPhaseCompleted {
		return fmt.Errorf("invalid phase for snapshot request %s, expected %s, got %s", snapshotRequestName, volumes.RequestPhaseCompleted, snapshotRequest.Status.Phase)
	}
	defer s.logger.Debugf("Reconciled completed volume snapshots request %s", snapshotRequestName)

	return nil
}
