package csi

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func (s *VolumeSnapshotter) reconcileCanceling(ctx context.Context, requestObj runtime.Object, requestName string, request *volumes.SnapshotsRequest, status *volumes.SnapshotsStatus) (retErr error) {
	s.logger.Debugf("Reconciling volume snapshots canceling for request %s", requestName)
	if status.Phase != volumes.RequestPhaseCanceling {
		return fmt.Errorf("invalid phase for snapshot request %s, expected %s, got %s", requestName, volumes.RequestPhaseCanceling, status.Phase)
	}
	s.logger.Debugf("Reconciled volume snapshots canceling for request %s", requestName)

	if len(request.Requests) == 0 {
		status.Phase = volumes.RequestPhaseCanceled
		s.logger.Debugf("Snapshot request %s does not contain any volume snapshots", requestName)
		return nil
	}

	s.eventRecorder.Eventf(requestObj, corev1.EventTypeWarning, "Canceling", "Canceling volume snapshots creation")
	stillCanceling := false
	defer func() {
		if retErr == nil {
			return
		}
		status.Phase = volumes.RequestPhaseFailed
		status.Error.Message = retErr.Error()
		s.eventRecorder.Eventf(requestObj, corev1.EventTypeWarning, "CancelingFailed", "Failed to cancel volume snapshots: %v", retErr)
	}()

	for _, volumeSnapshotRequest := range request.Requests {
		pvcName := types.NamespacedName{
			Namespace: volumeSnapshotRequest.PersistentVolumeClaim.Namespace,
			Name:      volumeSnapshotRequest.PersistentVolumeClaim.Name,
		}.String()
		snapshotStatus, ok := status.Snapshots[pvcName]
		if !ok {
			// the volume snapshot wasn't found
			continue
		}
		tryDeletingVolumeSnapshot := false
		if snapshotStatus.Phase == volumes.RequestPhaseNotStarted ||
			snapshotStatus.Phase == volumes.RequestPhaseInProgress ||
			snapshotStatus.Phase == volumes.RequestPhaseCompletedCleaningUp ||
			snapshotStatus.Phase == volumes.RequestPhaseCompleted {
			// When the volume snapshots phase is NotStarted or InProgress, it could mean that the
			// volume snapshot has been created, but the snapshot request has not been yet updated
			// to the new phase (Completed).
			//
			// When the volume snapshots phase is CompletedCleaningUp or Completed, it means that
			// the volume snapshot has been created.
			tryDeletingVolumeSnapshot = true
		}
		if !tryDeletingVolumeSnapshot {
			continue
		}

		// Update volume snapshot phase to Canceling
		snapshotStatus.Phase = volumes.RequestPhaseCanceling
		status.Snapshots[pvcName] = snapshotStatus
		volumeSnapshotName := fmt.Sprintf("%s-%s", volumeSnapshotRequest.PersistentVolumeClaim.Name, requestName)
		deleted, err := s.deleteVolumeSnapshot(ctx, volumeSnapshotRequest.PersistentVolumeClaim.Namespace, volumeSnapshotName)
		if err != nil {
			return fmt.Errorf("failed to delete volume snapshot: %w", err)
		}
		if deleted {
			snapshotStatus.Phase = volumes.RequestPhaseCanceled
			status.Snapshots[pvcName] = snapshotStatus
		} else {
			stillCanceling = true
		}
	}

	if !stillCanceling {
		status.Phase = volumes.RequestPhaseCanceled
		s.eventRecorder.Eventf(requestObj, corev1.EventTypeWarning, "Canceled", "Volume snapshots creation canceled")
	}
	return nil
}
