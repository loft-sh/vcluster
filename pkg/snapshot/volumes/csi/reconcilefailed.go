package csi

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
)

func (s *VolumeSnapshotter) reconcileFailed(ctx context.Context, snapshotRequestName string, snapshotRequest *volumes.SnapshotRequest) error {
	s.logger.Infof("Reconciling completed volume snapshots request %s", snapshotRequestName)
	if snapshotRequest.Status.Phase != volumes.RequestPhaseFailed {
		return fmt.Errorf("invalid phase for snapshot request %s, expected %s, got %s", snapshotRequestName, volumes.RequestPhaseFailed, snapshotRequest.Status.Phase)
	}
	defer s.logger.Infof("Reconciled completed volume snapshots request %s", snapshotRequestName)
	return nil
}
