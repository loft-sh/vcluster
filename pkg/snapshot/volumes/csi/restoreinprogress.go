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
	"k8s.io/utils/ptr"
)

func (r *Restorer) reconcileInProgress(ctx context.Context, requestObj runtime.Object, requestName string, request *volumes.RestoreRequestSpec, status *volumes.RestoreRequestStatus) (retErr error) {
	r.logger.Infof("Reconciling in-progress volumes restore request %s", requestName)
	if status.Phase != volumes.RequestPhaseInProgress {
		return fmt.Errorf("invalid phase for snapshot request %s, expected %s, got %s", requestName, volumes.RequestPhaseInProgress, status.Phase)
	}
	defer r.logger.Infof("Reconciled in-progress volumes restore request %s", requestName)

	defer func() {
		if retErr == nil {
			return
		}
		status.Phase = volumes.RequestPhaseFailed
		status.Error.Message = retErr.Error()
	}()

	hasInProgressRestores := false
	cleaningUpSnapshots := false
	hasCompletedRestores := false
	hasSkippedRestores := false
	failedRestoresCount := 0
	for _, volumeRestoreRequest := range request.Requests {
		pvcName := types.NamespacedName{
			Namespace: volumeRestoreRequest.PersistentVolumeClaim.Namespace,
			Name:      volumeRestoreRequest.PersistentVolumeClaim.Name,
		}.String()
		volumeRestoreStatus, ok := status.PersistentVolumeClaims[pvcName]
		if !ok {
			return fmt.Errorf("failed to find status for PVC %s in restore snapshot request %s", pvcName, requestName)
		}

		switch volumeRestoreStatus.Phase {
		case volumes.RequestPhaseNotStarted:
			volumeRestoreStatus.Phase = volumes.RequestPhaseInProgress
			fallthrough
		case volumes.RequestPhaseInProgress:
			newStatus, err := r.reconcileInProgressPVC(ctx, requestObj, requestName, volumeRestoreRequest, volumeRestoreStatus)
			status.PersistentVolumeClaims[pvcName] = newStatus
			if err != nil {
				r.logger.Errorf("failed to reconcile in-progress volumes restore request %s for PVC %s: %v", requestName, pvcName, err)
			}
			switch newStatus.Phase {
			case volumes.RequestPhaseInProgress:
				hasInProgressRestores = true
				continue
			case volumes.RequestPhaseCompletedCleaningUp:
				fallthrough
			case volumes.RequestPhaseFailedCleaningUp:
				cleaningUpSnapshots = true
			case volumes.RequestPhaseSkipped:
				hasSkippedRestores = true
			default:
				return fmt.Errorf("unexpected phase %s for restoring PVC %s", newStatus.Phase, pvcName)
			}
		case volumes.RequestPhaseCompletedCleaningUp:
			fallthrough
		case volumes.RequestPhaseFailedCleaningUp:
			if volumeRestoreStatus.Phase == volumes.RequestPhaseCompletedCleaningUp {
				// if the PVC has been re-created, wait for it to be bound
				pvc, err := r.kubeClient.CoreV1().
					PersistentVolumeClaims(volumeRestoreRequest.PersistentVolumeClaim.Namespace).
					Get(ctx, volumeRestoreRequest.PersistentVolumeClaim.Name, metav1.GetOptions{})
				if err != nil {
					volumeRestoreStatus.Phase = volumeRestoreStatus.Phase.Failed()
					checkErr := fmt.Errorf("failed to check PVC %s/%s: %w", volumeRestoreRequest.PersistentVolumeClaim.Namespace, volumeRestoreRequest.PersistentVolumeClaim.Name, err)
					r.logger.Errorf(checkErr.Error())
					volumeRestoreStatus.Error.Message = checkErr.Error()
					status.PersistentVolumeClaims[pvcName] = volumeRestoreStatus
					if volumeRestoreStatus.CleaningUp() {
						cleaningUpSnapshots = true
					} else {
						failedRestoresCount++
					}
					continue
				}
				if pvc.Status.Phase != corev1.ClaimBound {
					// PVC is not bound yet, don't clean up the VolumeSnapshot resource yet
					cleaningUpSnapshots = true
					continue
				}
			}
			volumeSnapshotName := fmt.Sprintf("%s-%s", volumeRestoreRequest.PersistentVolumeClaim.Name, requestName)
			cleanedUp, err := r.cleanupVolumeSnapshotResource(ctx, volumeRestoreRequest.PersistentVolumeClaim.Namespace, volumeSnapshotName)
			if err != nil {
				volumeRestoreStatus.Phase = volumeRestoreStatus.Phase.Failed()
				volumeRestoreStatus.Error.Message = fmt.Errorf("failed to cleanup volume snapshot resources: %w", err).Error()
				status.PersistentVolumeClaims[pvcName] = volumeRestoreStatus
				if volumeRestoreStatus.CleaningUp() {
					cleaningUpSnapshots = true
				} else {
					failedRestoresCount++
				}
				continue
			}
			if cleanedUp {
				volumeRestoreStatus.Phase = volumeRestoreStatus.Phase.Next()
				status.PersistentVolumeClaims[pvcName] = volumeRestoreStatus
				if volumeRestoreStatus.Phase == volumes.RequestPhaseFailed {
					failedRestoresCount++
				} else if volumeRestoreStatus.Phase == volumes.RequestPhaseCompleted {
					hasCompletedRestores = true
				}
			} else {
				cleaningUpSnapshots = true
			}
		case volumes.RequestPhaseCompleted:
			hasCompletedRestores = true
			r.logger.Debugf("PVC %s has been already restored", pvcName)
		case volumes.RequestPhaseSkipped:
			hasSkippedRestores = true
			r.logger.Debugf("PVC %s already exists, restore skipped", pvcName)
		case volumes.RequestPhaseFailed:
			failedRestoresCount++
			r.logger.Errorf("Failed to restore PVC %s", pvcName)
		default:
			return fmt.Errorf("invalid restore request phase %s for PVC %s in restore snapshot request %s", volumeRestoreStatus.Phase, pvcName, requestName)
		}
	}

	hasFailedRestores := failedRestoresCount > 0
	if hasInProgressRestores || cleaningUpSnapshots {
		status.Phase = volumes.RequestPhaseInProgress
	} else if hasCompletedRestores && hasFailedRestores {
		status.Phase = volumes.RequestPhasePartiallyFailed
		status.Error.Message = fmt.Sprintf("%d out of %d PVCs failed to restore", failedRestoresCount, len(request.Requests))
	} else if hasCompletedRestores {
		status.Phase = volumes.RequestPhaseCompleted
	} else if hasFailedRestores {
		status.Phase = volumes.RequestPhaseFailed
		if hasSkippedRestores {
			status.Error.Message = "some PVC restores have failed, others have been skipped"
		} else {
			status.Error.Message = "all PVC restores have failed"
		}
	} else if hasSkippedRestores {
		status.Phase = volumes.RequestPhaseSkipped
	} else {
		return fmt.Errorf("unexpected state for restore request %s, expected at least 1 volume restore to be in progress, cleaning up, completed or failed", requestName)
	}

	return nil
}

func (r *Restorer) reconcileInProgressPVC(ctx context.Context, requestObj runtime.Object, requestName string, volumeRestoreRequest volumes.RestoreRequest, volumeRestoreStatus volumes.RestoreStatus) (status volumes.RestoreStatus, retErr error) {
	if volumeRestoreStatus.Phase != volumes.RequestPhaseInProgress {
		return volumeRestoreStatus, fmt.Errorf("invalid phase for snapshot request %s, expected %s, got %s", requestName, volumes.RequestPhaseInProgress, volumeRestoreStatus.Phase)
	}
	status = volumeRestoreStatus
	defer func() {
		if retErr != nil {
			status.Phase = volumes.RequestPhaseFailedCleaningUp
		}
		r.inProgressPVCReconcileFinished(requestObj, volumeRestoreRequest, status, retErr)
	}()

	// First, check if the PVC already exists
	originalPVC := &volumeRestoreRequest.PersistentVolumeClaim
	_, err := r.kubeClient.CoreV1().PersistentVolumeClaims(originalPVC.Namespace).Get(ctx, originalPVC.Name, metav1.GetOptions{})
	if err == nil {
		// existing PVC found
		status.Phase = volumes.RequestPhaseSkipped
		return status, nil
	} else if !kerrors.IsNotFound(err) {
		return status, fmt.Errorf("failed to get PVC %s/%s: %w", originalPVC.Namespace, originalPVC.Name, err)
	}

	// PVC hasn't been found, restore it from VolumeSnapshot. For this we need pre-provisioned VolumeSnapshot
	// and VolumeSnapshotContent resources.

	volumeSnapshotName := fmt.Sprintf("%s-%s", volumeRestoreRequest.PersistentVolumeClaim.Name, requestName)
	pvcName := types.NamespacedName{
		Namespace: volumeRestoreRequest.PersistentVolumeClaim.Namespace,
		Name:      volumeRestoreRequest.PersistentVolumeClaim.Name,
	}

	// Check if the pre-provisioned VolumeSnapshotContent resource exists. If it doesn't, create it.
	justCreated := false
	snapshotRequest := volumes.SnapshotRequest{
		PersistentVolumeClaim:   volumeRestoreRequest.PersistentVolumeClaim,
		CSIDriver:               volumeRestoreRequest.CSIDriver,
		VolumeSnapshotClassName: volumeRestoreRequest.VolumeSnapshotClassName,
	}
	volumeSnapshotContent, err := r.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Get(ctx, volumeSnapshotName, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		// create new pre-provisioned VolumeSnapshotContent
		volumeSnapshotContent, err = r.createVolumeSnapshotContentResource(
			ctx,
			constants.RestoreRequestLabel,
			requestName,
			snapshotRequest,
			volumeRestoreRequest.SnapshotHandle,
			snapshotsv1api.VolumeSnapshotContentRetain)
		if err != nil {
			return status, fmt.Errorf("failed to create VolumeSnapshotContent for the PersistentVolumeClaim %s: %w", pvcName, err)
		}
		justCreated = true
	} else if err != nil {
		return status, fmt.Errorf("failed to get VolumeSnapshotContent %s: %w", volumeSnapshotContent.Name, err)
	}

	// Check if the pre-provisioned VolumeSnapshot resource exists. If it doesn't, create it.
	volumeSnapshot, err := r.snapshotsClient.SnapshotV1().VolumeSnapshots(pvcName.Namespace).Get(ctx, volumeSnapshotName, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		// create new VolumeSnapshot
		volumeSnapshot, err = r.createPreProvisionedVolumeSnapshot(ctx, constants.RestoreRequestLabel, requestName, snapshotRequest)
		if err != nil {
			return status, fmt.Errorf("failed to create VolumeSnapshot for the PersistentVolumeClaim %s: %w", pvcName, err)
		}
		justCreated = true
	} else if err != nil {
		return status, fmt.Errorf("failed to get VolumeSnapshot %s/%s: %w", volumeSnapshot.Namespace, volumeSnapshot.Name, err)
	}

	if justCreated {
		// wait for pre-provisioned VolumeSnapshot and VolumeSnapshotContent to be ready
		return status, nil
	}

	if volumeSnapshot.Status == nil {
		return status, nil
	}

	// check if VolumeSnapshot has failed
	if volumeSnapshot.Status.Error != nil {
		var errorMessage string
		if volumeSnapshot.Status.Error.Message != nil {
			errorMessage = fmt.Sprintf(
				"VolumeSnapshot %s/%s (for PersistentVolumeClaim %s) has a status error message %s",
				volumeSnapshot.Namespace,
				volumeSnapshot.Name,
				pvcName.String(),
				*volumeSnapshot.Status.Error.Message)
		} else {
			errorMessage = fmt.Sprintf(
				"VolumeSnapshot %s/%s (for PersistentVolumeClaim %s) has failed with an unknown error",
				volumeSnapshot.Namespace,
				volumeSnapshot.Name,
				pvcName.String())
		}

		return status, errors.New(errorMessage)
	}

	// check if VolumeSnapshot is ready
	if volumeSnapshot.Status.ReadyToUse == nil || !*volumeSnapshot.Status.ReadyToUse {
		// VolumeSnapshot is still not ready
		return status, nil
	}

	if volumeSnapshotContent.Status == nil {
		return status, nil
	}

	// check if VolumeSnapshotContent has failed
	if volumeSnapshotContent.Status.Error != nil {
		var errorMessage string
		if volumeSnapshotContent.Status.Error.Message != nil {
			errorMessage = fmt.Sprintf(
				"VolumeSnapshotContent %s (for PersistentVolumeClaim %s) has a status error message: %s",
				volumeSnapshotContent.Name,
				pvcName.String(),
				*volumeSnapshotContent.Status.Error.Message)
		} else {
			errorMessage = fmt.Sprintf(
				"VolumeSnapshotContent %s (for PersistentVolumeClaim %s) has failed with an unknown error",
				volumeSnapshotContent.Name,
				pvcName.String())
		}
		return status, errors.New(errorMessage)
	}

	// check if VolumeSnapshotContent is ready
	if volumeSnapshotContent.Status.ReadyToUse == nil || !*volumeSnapshotContent.Status.ReadyToUse {
		// VolumeSnapshotContent is still not ready
		return status, nil
	}

	// both VolumeSnapshot and VolumeSnapshotContent are ready, now we can restore the PVC
	originalPersistentVolumeClaim := volumeRestoreRequest.PersistentVolumeClaim
	delete(originalPersistentVolumeClaim.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
	delete(originalPersistentVolumeClaim.Annotations, "pv.kubernetes.io/bind-completed")
	delete(originalPersistentVolumeClaim.Annotations, "pv.kubernetes.io/bound-by-controller")
	delete(originalPersistentVolumeClaim.Annotations, "volume.beta.kubernetes.io/storage-provisioner")
	delete(originalPersistentVolumeClaim.Annotations, "volume.kubernetes.io/storage-provisioner")

	wantedRestoredPersistentVolumeClaim := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        originalPersistentVolumeClaim.Name,
			Namespace:   originalPersistentVolumeClaim.Namespace,
			Annotations: originalPersistentVolumeClaim.Annotations,
			Labels:      originalPersistentVolumeClaim.Labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      originalPersistentVolumeClaim.Spec.AccessModes,
			Selector:         originalPersistentVolumeClaim.Spec.Selector,
			Resources:        originalPersistentVolumeClaim.Spec.Resources,
			StorageClassName: originalPersistentVolumeClaim.Spec.StorageClassName,
			VolumeMode:       originalPersistentVolumeClaim.Spec.VolumeMode,
			DataSource: &corev1.TypedLocalObjectReference{
				APIGroup: ptr.To(snapshotsv1api.GroupName),
				Kind:     "VolumeSnapshot",
				Name:     volumeSnapshot.Name,
			},
			DataSourceRef:             nil,
			VolumeAttributesClassName: nil,
		},
	}
	restoredPersistentVolumeClaim, err := r.kubeClient.CoreV1().
		PersistentVolumeClaims(volumeSnapshot.Namespace).
		Create(ctx, wantedRestoredPersistentVolumeClaim, metav1.CreateOptions{})
	if err != nil {
		return status, fmt.Errorf(
			"failed to create PersistentVolumeClaim %s/%s from VolumeSnapshot %s/%s: %w",
			originalPersistentVolumeClaim.Namespace,
			originalPersistentVolumeClaim.Name,
			volumeSnapshot.Namespace,
			volumeSnapshot.Name,
			err)
	}

	status.Phase = volumes.RequestPhaseCompletedCleaningUp
	r.logger.Infof(
		"Restored PersistentVolumeClaim %s/%s from VolumeSnapshot %s/%s",
		restoredPersistentVolumeClaim.Namespace, restoredPersistentVolumeClaim.Name,
		volumeSnapshot.Namespace, volumeSnapshot.Name)

	return status, nil
}

func (r *Restorer) inProgressPVCReconcileFinished(requestObj runtime.Object, volumeRestoreRequest volumes.RestoreRequest, volumeRestoreStatus volumes.RestoreStatus, err error) {
	var eventType, reason, messageFmt string
	var args []interface{}

	switch volumeRestoreStatus.Phase {
	case volumes.RequestPhaseCompleted:
		eventType = corev1.EventTypeNormal
		reason = "VolumeRestored"
		messageFmt = "Restored PersistentVolumeClaim %s/%s from volume snapshot with handle %s"
		args = []interface{}{
			volumeRestoreRequest.PersistentVolumeClaim.Namespace,
			volumeRestoreRequest.PersistentVolumeClaim.Name,
			volumeRestoreRequest.SnapshotHandle,
		}
	case volumes.RequestPhaseFailed:
		eventType = corev1.EventTypeWarning
		reason = "VolumeRestoreFailed"
		messageFmt = "Failed to restore PersistentVolumeClaim %s/%s: %v"
		args = []interface{}{
			volumeRestoreRequest.PersistentVolumeClaim.Namespace,
			volumeRestoreRequest.PersistentVolumeClaim.Name,
			err,
		}
	case volumes.RequestPhaseSkipped:
		eventType = corev1.EventTypeNormal
		reason = "VolumeRestoreSkipped"
		messageFmt = "Skipped restoring PersistentVolumeClaim %s/%s"
		args = []interface{}{
			volumeRestoreRequest.PersistentVolumeClaim.Namespace,
			volumeRestoreRequest.PersistentVolumeClaim.Name,
		}
	default:
		return
	}

	r.eventRecorder.Eventf(requestObj, eventType, reason, messageFmt, args...)
}
