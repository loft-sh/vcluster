package csi

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func (s *VolumeSnapshotter) reconcileDeleting(ctx context.Context, requestObj runtime.Object, requestName string, request *volumes.SnapshotsRequest, status *volumes.SnapshotsStatus) (retErr error) {
	if !status.IsDeletingVolumeSnapshots() {
		return fmt.Errorf("invalid phase for snapshot request %s, expected %s or %s, got %s", requestName, volumes.RequestPhaseDeleting, volumes.RequestPhaseCanceling, status.Phase)
	}
	s.logger.Debugf("Reconciling volume snapshots %s for request %s", status.Phase, requestName)
	defer s.logger.Debugf("Reconciled volume snapshots %s for request %s", status.Phase, requestName)

	if len(request.Requests) == 0 {
		status.Phase = status.Phase.Next()
		s.logger.Debugf("Snapshot request %s does not contain any volume snapshots", requestName)
		return nil
	}

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

		volumeSnapshotName := fmt.Sprintf("%s-%s", volumeSnapshotRequest.PersistentVolumeClaim.Name, requestName)
		if volumeSnapshotStatus.IsVolumeSnapshotMaybeCreated() || volumeSnapshotStatus.IsDeletingVolumeSnapshot() {
			// Re-create VolumeSnapshot and VolumeSnapshotContent resources if the following conditions are met:
			// - this is the Deletion request
			// - volume snapshot is already being deleted (because then, if VolumeSnapshot and VolumeSnapshotContent
			//   resources are not found, it means that they have been already deleted)
			recreateResourceIfNotFound :=
				status.RecreateVolumeSnapshotsWhenDeleting() &&
					!volumeSnapshotStatus.IsDeletingVolumeSnapshot()
			deletedResources, err := s.deleteVolumeSnapshot(
				ctx,
				constants.SnapshotRequestLabel,
				requestName,
				volumeSnapshotRequest,
				volumeSnapshotStatus.SnapshotHandle,
				recreateResourceIfNotFound)
			// check for errors
			if err != nil {
				return fmt.Errorf("failed to delete volume snapshot %s: %w", volumeSnapshotName, err)
			}
			if volumeSnapshotStatus.Phase != status.Phase {
				// update volume status to Canceling / Deleting
				volumeSnapshotStatus.Phase = status.Phase
				status.Snapshots[pvcName] = volumeSnapshotStatus
				s.eventRecorder.Eventf(
					requestObj,
					corev1.EventTypeNormal,
					string(status.Phase),
					"%s volume snapshot for PVC %s/%s",
					status.Phase,
					volumeSnapshotRequest.PersistentVolumeClaim.Namespace,
					volumeSnapshotRequest.PersistentVolumeClaim.Name)
			}
			if deletedResources {
				// resources deleted, so just update the status
				volumeSnapshotStatus.Phase = volumeSnapshotStatus.Phase.Next()
				status.Snapshots[pvcName] = volumeSnapshotStatus
				s.eventRecorder.Eventf(
					requestObj,
					corev1.EventTypeNormal,
					string(volumeSnapshotStatus.Phase),
					"%s volume snapshot for PVC %s/%s",
					status.Phase,
					volumeSnapshotRequest.PersistentVolumeClaim.Namespace,
					volumeSnapshotRequest.PersistentVolumeClaim.Name)
			} else {
				stillDeleting = true
			}
		}
	}

	if !stillDeleting {
		status.Phase = status.Phase.Next()
		s.eventRecorder.Eventf(requestObj, corev1.EventTypeNormal, string(status.Phase), "%s volume snapshots", status.Phase)
	}
	return nil
}
