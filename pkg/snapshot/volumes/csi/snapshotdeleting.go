package csi

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func (s *VolumeSnapshotter) reconcileDeleting(ctx context.Context, requestObj runtime.Object, requestName string, request *volumes.SnapshotsRequest, status *volumes.SnapshotsStatus) (retErr error) {
	if !status.DeletingVolumeSnapshots() {
		return fmt.Errorf("invalid phase for snapshot request %s, expected %s or %s, got %s", requestName, volumes.RequestPhaseDeleting, volumes.RequestPhaseCanceling, status.Phase)
	}
	s.logger.Debugf("Reconciling volume snapshots %s for request %s", status.Phase, requestName)
	defer s.logger.Debugf("Reconciled volume snapshots %s for request %s", status.Phase, requestName)

	if len(request.Requests) == 0 {
		status.Phase = status.Phase.Next()
		s.logger.Debugf("Snapshot request %s does not contain any volume snapshots", requestName)
		return nil
	}

	s.eventRecorder.Eventf(requestObj, corev1.EventTypeNormal, string(status.Phase), "%s volume snapshots", status.Phase)
	stillDeleting := false
	defer func() {
		if retErr == nil {
			return
		}
		status.Phase = volumes.RequestPhaseFailed
		status.Error.Message = retErr.Error()
		s.eventRecorder.Eventf(requestObj, corev1.EventTypeWarning, fmt.Sprintf("%sFailed", status.Phase), "%s volume snapshots failed: %v", status.Phase, retErr)
	}()

	for _, volumeSnapshotRequest := range request.Requests {
		pvcName := types.NamespacedName{
			Namespace: volumeSnapshotRequest.PersistentVolumeClaim.Namespace,
			Name:      volumeSnapshotRequest.PersistentVolumeClaim.Name,
		}.String()
		volumeSnapshotStatus, ok := status.Snapshots[pvcName]
		if !ok {
			// the volume snapshot wasn't found
			continue
		}
		if volumeSnapshotStatus.DeletingVolumeSnapshot() {
			stillDeleting = true
			continue
		}
		tryDeletingVolumeSnapshot := false
		if volumeSnapshotStatus.Phase == volumes.RequestPhaseNotStarted ||
			volumeSnapshotStatus.Phase == volumes.RequestPhaseInProgress ||
			volumeSnapshotStatus.Phase == volumes.RequestPhaseCompletedCleaningUp ||
			volumeSnapshotStatus.Phase == volumes.RequestPhaseCompleted {
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

		// Update the volume snapshot phase to Deleting/Canceling
		volumeSnapshotStatus.Phase = status.Phase
		status.Snapshots[pvcName] = volumeSnapshotStatus
		volumeSnapshotName := fmt.Sprintf("%s-%s", volumeSnapshotRequest.PersistentVolumeClaim.Name, requestName)
		deleted, err := s.deleteVolumeSnapshot(ctx, volumeSnapshotRequest.PersistentVolumeClaim.Namespace, volumeSnapshotName)
		if err != nil {
			return fmt.Errorf("failed to delete volume snapshot: %w", err)
		}
		if deleted {
			volumeSnapshotStatus.Phase = volumeSnapshotStatus.Phase.Next()
			status.Snapshots[pvcName] = volumeSnapshotStatus
		} else {
			stillDeleting = true
		}
	}

	if !stillDeleting {
		status.Phase = status.Phase.Next()
		s.eventRecorder.Eventf(requestObj, corev1.EventTypeNormal, string(status.Phase), "%s volume snapshots", status.Phase)
	}
	return nil
}
