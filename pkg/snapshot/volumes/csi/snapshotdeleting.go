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
		if volumeSnapshotStatus.IsVolumeSnapshotMaybeCreated() {
			// The volume snapshot could have been created, the deletion has not been started, so
			// trigger deletion here.
			deleted, err := s.deleteVolumeSnapshot(
				ctx,
				constants.SnapshotRequestLabel,
				requestName,
				volumeSnapshotRequest,
				volumeSnapshotStatus.SnapshotHandle,
				status.RecreateVolumeSnapshotsWhenDeleting())
			// check for errors
			if err != nil {
				return fmt.Errorf("failed to delete volume snapshot %s: %w", volumeSnapshotName, err)
			}
			volumeSnapshotStatus.Phase = status.Phase
			status.Snapshots[pvcName] = volumeSnapshotStatus
			stillDeleting = !deleted
			s.eventRecorder.Eventf(
				requestObj,
				corev1.EventTypeNormal,
				string(status.Phase),
				"%s volume snapshot for PVC %s/%s",
				status.Phase,
				volumeSnapshotRequest.PersistentVolumeClaim.Namespace,
				volumeSnapshotRequest.PersistentVolumeClaim.Name)
		} else if volumeSnapshotStatus.IsDeletingVolumeSnapshot() {
			// Volume snapshot deletion has been already started, which means that the resources
			// have been already re-created if needed. Therefore, just check if the resources have
			// been already deleted.
			var volumeSnapshotContentName string
			if volumeSnapshotStatus.RecreateVolumeSnapshotWhenDeleting() {
				// When the VolumeSnapshot and VolumeSnapshotContent are re-created, it means that
				// they are pre-provisioned, so the VolumeSnapshotContent name is manually set to
				// the same name as the VolumeSnapshot name.
				volumeSnapshotContentName = volumeSnapshotName
			}
			volumeSnapshotExists, volumeSnapshotContentExists, err := s.checkIfVolumeSnapshotResourcesExist(
				ctx,
				volumeSnapshotRequest.PersistentVolumeClaim.Namespace,
				volumeSnapshotName,
				volumeSnapshotContentName)
			if err != nil {
				return fmt.Errorf("failed to check if volume snapshot resources exist: %w", err)
			}
			if !volumeSnapshotExists && !volumeSnapshotContentExists {
				volumeSnapshotStatus.Phase = volumeSnapshotStatus.Phase.Next()
				status.Snapshots[pvcName] = volumeSnapshotStatus
			} else {
				if volumeSnapshotExists {
					s.logger.Debugf(
						"VolumeSnapshot %s for PVC %s/%s is still being deleted",
						volumeSnapshotName,
						volumeSnapshotRequest.PersistentVolumeClaim.Namespace,
						volumeSnapshotRequest.PersistentVolumeClaim.Name)
				}
				if volumeSnapshotContentExists {
					s.logger.Debugf("VolumeSnapshotContent %s is still being deleted", volumeSnapshotContentName)
				}
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
