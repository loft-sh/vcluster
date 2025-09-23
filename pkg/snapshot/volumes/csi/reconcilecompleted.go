package csi

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
)

func (s *VolumeSnapshotter) reconcileCompleted(_ context.Context, requestName string, request *volumes.SnapshotsRequest, status *volumes.SnapshotsStatus) error {
	s.logger.Debugf("Reconciling completed volume snapshots request %s", requestName)
	if status.Phase != volumes.RequestPhaseCompleted {
		return fmt.Errorf("invalid phase for snapshot request %s, expected %s, got %s", requestName, volumes.RequestPhaseCompleted, status.Phase)
	}
	defer s.logger.Debugf("Reconciled completed volume snapshots request %s", requestName)

	return nil
}
