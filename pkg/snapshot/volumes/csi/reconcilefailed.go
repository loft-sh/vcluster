package csi

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
)

func (s *VolumeSnapshotter) reconcileFailed(ctx context.Context, requestName string, request *volumes.SnapshotsRequest, status *volumes.SnapshotsStatus) error {
	s.logger.Infof("Reconciling completed volume snapshots request %s", requestName)
	if status.Phase != volumes.RequestPhaseFailed {
		return fmt.Errorf("invalid phase for snapshot request %s, expected %s, got %s", requestName, volumes.RequestPhaseFailed, status.Phase)
	}
	defer s.logger.Infof("Reconciled completed volume snapshots request %s", requestName)
	return nil
}
