package csi

import (
	"context"
	"errors"
	"fmt"

	snapshotsv1api "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func (s *VolumeSnapshotter) reconcileInProgress(ctx context.Context, requestObj runtime.Object, requestName string, request *volumes.SnapshotsRequest, status *volumes.SnapshotsStatus) (retErr error) {
	s.logger.Debugf("Reconciling in-progress volume snapshots request %s", requestName)
	if status.Phase != volumes.RequestPhaseInProgress {
		return fmt.Errorf("invalid phase for snapshot request %s, expected %s, got %s", requestName, volumes.RequestPhaseInProgress, status.Phase)
	}
	defer s.logger.Debugf("Reconciled in-progress volume snapshots request %s", requestName)

	if len(request.Requests) == 0 {
		status.Phase = volumes.RequestPhaseCompleted
		s.logger.Infof("Snapshot request %s does not contain any volume snapshots", requestName)
		return nil
	}

	hasInProgressSnapshots := false
	cleaningUpSnapshots := false
	hasCompletedSnapshots := false
	failedSnapshotsCount := 0
	defer func() {
		if retErr == nil {
			return
		}
		status.Phase = volumes.RequestPhaseFailed
		status.Error.Message = retErr.Error()
		s.eventRecorder.Eventf(requestObj, corev1.EventTypeWarning, "VolumeSnapshotsFailed", "Failed to create volume snapshots: %v", retErr)
	}()

	if status.Snapshots == nil {
		status.Snapshots = map[string]volumes.SnapshotStatus{}
	}
	for _, volumeSnapshotRequest := range request.Requests {
		pvcName := types.NamespacedName{
			Namespace: volumeSnapshotRequest.PersistentVolumeClaim.Namespace,
			Name:      volumeSnapshotRequest.PersistentVolumeClaim.Name,
		}.String()
		snapshotStatus, ok := status.Snapshots[pvcName]
		if !ok {
			snapshotStatus = volumes.SnapshotStatus{
				Phase: volumes.RequestPhaseInProgress,
			}
			status.Snapshots[pvcName] = snapshotStatus
		}

		switch snapshotStatus.Phase {
		case volumes.RequestPhaseNotStarted:
			snapshotStatus.Phase = volumes.RequestPhaseInProgress
			fallthrough
		case volumes.RequestPhaseInProgress:
			newStatus := s.reconcileInProgressPVC(ctx, requestObj, requestName, volumeSnapshotRequest, snapshotStatus)
			status.Snapshots[pvcName] = newStatus
			if newStatus.Phase == volumes.RequestPhaseInProgress {
				// snapshot creation is still in progress
				hasInProgressSnapshots = true
			} else if newStatus.CleaningUp() {
				cleaningUpSnapshots = true
			}
		case volumes.RequestPhaseCompletedCleaningUp:
			fallthrough
		case volumes.RequestPhaseFailedCleaningUp:
			volumeSnapshotName := fmt.Sprintf("%s-%s", volumeSnapshotRequest.PersistentVolumeClaim.Name, requestName)
			cleanedUp, err := s.cleanupVolumeSnapshotResource(ctx, volumeSnapshotRequest.PersistentVolumeClaim.Namespace, volumeSnapshotName)
			if err != nil {
				snapshotStatus.Phase = snapshotStatus.Phase.Failed()
				snapshotStatus.Error.Message = fmt.Errorf("failed to cleanup volume snapshot resources: %w", err).Error()
				status.Snapshots[pvcName] = snapshotStatus
				if snapshotStatus.CleaningUp() {
					cleaningUpSnapshots = true
				} else {
					failedSnapshotsCount++
				}
				continue
			}
			if cleanedUp {
				snapshotStatus.Phase = snapshotStatus.Phase.Next()
				status.Snapshots[pvcName] = snapshotStatus
				if snapshotStatus.Phase == volumes.RequestPhaseFailed {
					failedSnapshotsCount++
				} else if snapshotStatus.Phase == volumes.RequestPhaseCompleted {
					hasCompletedSnapshots = true
				}
			} else {
				cleaningUpSnapshots = true
			}
		case volumes.RequestPhaseCompleted:
			hasCompletedSnapshots = true
		case volumes.RequestPhaseFailed:
			failedSnapshotsCount++
		default:
			return fmt.Errorf("invalid snapshot request phase %s for PVC %s in volume snapshot request %s", snapshotStatus.Phase, pvcName, requestName)
		}
	}

	hasFailedSnapshots := failedSnapshotsCount > 0
	if hasInProgressSnapshots || cleaningUpSnapshots {
		status.Phase = volumes.RequestPhaseInProgress
	} else if hasCompletedSnapshots && hasFailedSnapshots {
		status.Phase = volumes.RequestPhasePartiallyFailed
		status.Error.Message = fmt.Sprintf("%d out of %d volume snapshots have failed", failedSnapshotsCount, len(request.Requests))
		s.eventRecorder.Eventf(requestObj, corev1.EventTypeWarning, "VolumeSnapshotsPartiallyFailed", status.Error.Message)
	} else if hasCompletedSnapshots {
		status.Phase = volumes.RequestPhaseCompleted
	} else if hasFailedSnapshots {
		status.Phase = volumes.RequestPhaseFailed
		status.Error.Message = "all volume snapshots have failed"
		s.eventRecorder.Eventf(requestObj, corev1.EventTypeWarning, "VolumeSnapshotsFailed", status.Error.Message)
	} else {
		return fmt.Errorf("unexpected state for snapshot request %s, expected at least 1 snapshot to be in progress, completed or failed", requestName)
	}
	return nil
}

func (s *VolumeSnapshotter) reconcileInProgressPVC(ctx context.Context, requestObj runtime.Object, requestName string, volumeSnapshotRequest volumes.SnapshotRequest, volumeSnapshotStatus volumes.SnapshotStatus) volumes.SnapshotStatus {
	updatedStatus := func(err error) volumes.SnapshotStatus {
		if err != nil {
			volumeSnapshotStatus.Phase = volumes.RequestPhaseFailedCleaningUp
			volumeSnapshotStatus.Error.Message = err.Error()
		}
		s.inProgressPVCReconcileFinished(requestObj, volumeSnapshotRequest, volumeSnapshotStatus, err)
		return volumeSnapshotStatus
	}

	pvcName := types.NamespacedName{
		Namespace: volumeSnapshotRequest.PersistentVolumeClaim.Namespace,
		Name:      volumeSnapshotRequest.PersistentVolumeClaim.Name,
	}
	if volumeSnapshotStatus.Phase != volumes.RequestPhaseInProgress {
		return updatedStatus(fmt.Errorf("invalid volume snapshot request phase %s for PVC %s, expected %s, got %s", volumeSnapshotStatus.Phase, pvcName.String(), volumes.RequestPhaseInProgress, volumeSnapshotStatus.Phase))
	}

	// Check if VolumeSnapshot has been created
	volumeSnapshotName := fmt.Sprintf("%s-%s", volumeSnapshotRequest.PersistentVolumeClaim.Name, requestName)
	volumeSnapshot, err := s.snapshotsClient.SnapshotV1().VolumeSnapshots(pvcName.Namespace).Get(ctx, volumeSnapshotName, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return updatedStatus(fmt.Errorf("failed to get VolumeSnapshot %s/%s: %w", volumeSnapshot.Namespace, volumeSnapshot.Name, err))
	} else if kerrors.IsNotFound(err) {
		// create new VolumeSnapshot
		_, err = s.createVolumeSnapshotResource(ctx, requestName, volumeSnapshotName, pvcName, volumeSnapshotRequest.VolumeSnapshotClassName)
		return updatedStatus(err)
	}

	if volumeSnapshot.Status == nil {
		// VolumeSnapshot is still not ready
		return volumeSnapshotStatus
	}

	// check if VolumeSnapshot has failed
	if volumeSnapshot.Status.Error != nil {
		// VolumeSnapshot has failed
		var errorMessage string
		if volumeSnapshot.Status.Error.Message != nil {
			errorMessage = *volumeSnapshot.Status.Error.Message
		} else {
			errorMessage = fmt.Sprintf(
				"VolumeSnapshot %s/%s (for PersistentVolumeClaim %s) has failed with an unknown error",
				volumeSnapshot.Namespace,
				volumeSnapshot.Name,
				pvcName.String())
		}

		return updatedStatus(errors.New(errorMessage))
	}

	// check if VolumeSnapshot is ready
	if volumeSnapshot.Status.ReadyToUse == nil || !*volumeSnapshot.Status.ReadyToUse {
		// VolumeSnapshot is still not ready
		return volumeSnapshotStatus
	}

	// VolumeSnapshot is ready -> get VolumeSnapshotContents
	volumeSnapshotContentName := volumeSnapshot.Status.BoundVolumeSnapshotContentName
	if volumeSnapshotContentName == nil || *volumeSnapshotContentName == "" {
		return updatedStatus(fmt.Errorf("VolumeSnapshot %s/%s does not have bound VolumeSnapshotContent name set", volumeSnapshot.Namespace, volumeSnapshot.Name))
	}
	volumeSnapshotContent, err := s.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Get(ctx, *volumeSnapshotContentName, metav1.GetOptions{})
	if err != nil {
		return updatedStatus(fmt.Errorf("could not get bound VolumeSnapshotContent '%s' for VolumeSnapshot '%s': %w", *volumeSnapshotContentName, volumeSnapshot.Name, err))
	}

	if volumeSnapshotContent.Status == nil {
		// VolumeSnapshotContent is still not ready
		return volumeSnapshotStatus
	}

	// check if VolumeSnapshotContent has failed
	if volumeSnapshotContent.Status.Error != nil {
		// volumeSnapshotContent has failed
		var errorMessage string
		if volumeSnapshotContent.Status.Error.Message != nil {
			errorMessage = *volumeSnapshotContent.Status.Error.Message
		} else {
			errorMessage = fmt.Sprintf(
				"VolumeSnapshotContent %s (for PersistentVolumeClaim %s) has failed with an unknown error",
				volumeSnapshotContent.Name,
				pvcName.String())
		}
		return updatedStatus(errors.New(errorMessage))
	}

	// check if VolumeSnapshotContent is ready
	if volumeSnapshotContent.Status.ReadyToUse == nil || !*volumeSnapshotContent.Status.ReadyToUse {
		// VolumeSnapshotContent is still not ready
		return volumeSnapshotStatus
	}

	// VolumeSnapshotContent is ready -> read the snapshot handle
	if volumeSnapshotContent.Status.SnapshotHandle == nil {
		return updatedStatus(fmt.Errorf("VolumeSnapshotContent %s (for PersistentVolumeClaim %s) does not have status.snapshotHandle set", volumeSnapshotContent.Name, pvcName.String()))
	}
	volumeSnapshotStatus.SnapshotHandle = *volumeSnapshotContent.Status.SnapshotHandle
	volumeSnapshotStatus.Phase = volumes.RequestPhaseCompletedCleaningUp
	return volumeSnapshotStatus
}

func (s *VolumeSnapshotter) createVolumeSnapshotResource(ctx context.Context, requestName, volumeSnapshotName string, pvcName types.NamespacedName, volumeSnapshotClassName string) (*snapshotsv1api.VolumeSnapshot, error) {
	s.logger.Debugf("Create VolumeSnapshot %s for PersistentVolumeClaim %s for snapshot request %s", volumeSnapshotName, pvcName.String(), requestName)

	volumeSnapshot := &snapshotsv1api.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pvcName.Namespace,
			Name:      volumeSnapshotName,
			Labels: map[string]string{
				constants.SnapshotRequestLabel: requestName,
				persistentVolumeClaimNameLabel: pvcName.Name,
			},
		},
		Spec: snapshotsv1api.VolumeSnapshotSpec{
			Source: snapshotsv1api.VolumeSnapshotSource{
				PersistentVolumeClaimName: &pvcName.Name,
			},
		},
	}
	if volumeSnapshotClassName != "" {
		volumeSnapshot.Spec.VolumeSnapshotClassName = &volumeSnapshotClassName
	}

	var err error
	volumeSnapshot, err = s.snapshotsClient.SnapshotV1().VolumeSnapshots(pvcName.Namespace).Create(ctx, volumeSnapshot, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not create VolumeSnapshot resource for the PersistentVolumeClaim %s: %w", pvcName, err)
	}
	s.logger.Infof("Created VolumeSnapshot resource %s/%s for the PersistentVolumeClaim %s", volumeSnapshot.Namespace, volumeSnapshot.Name, pvcName)

	return volumeSnapshot, nil
}

func (s *VolumeSnapshotter) inProgressPVCReconcileFinished(requestObj runtime.Object, volumeSnapshotRequest volumes.SnapshotRequest, volumeSnapshotStatus volumes.SnapshotStatus, err error) {
	var eventType, reason, messageFmt string
	var args []interface{}

	switch volumeSnapshotStatus.Phase {
	case volumes.RequestPhaseCompleted:
		eventType = corev1.EventTypeNormal
		reason = "VolumeSnapshotCreated"
		messageFmt = "Created volume snapshot for PVC %s/%s, snapshot handle is %s"
		args = []interface{}{
			volumeSnapshotRequest.PersistentVolumeClaim.Namespace,
			volumeSnapshotRequest.PersistentVolumeClaim.Name,
			volumeSnapshotStatus.SnapshotHandle,
		}
	case volumes.RequestPhaseFailed:
		eventType = corev1.EventTypeWarning
		reason = "VolumeSnapshotFailed"
		messageFmt = "Failed to create volume snapshot for PVC %s/%s: %v"
		args = []interface{}{
			volumeSnapshotRequest.PersistentVolumeClaim.Namespace,
			volumeSnapshotRequest.PersistentVolumeClaim.Name,
			err,
		}
	default:
		return
	}

	s.eventRecorder.Eventf(requestObj, eventType, reason, messageFmt, args...)
}
