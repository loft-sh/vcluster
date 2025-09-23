package csi

import (
	"context"
	"errors"
	"fmt"

	snapshotsv1api "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	"github.com/loft-sh/vcluster/pkg/snapshot/meta"
	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (s *VolumeSnapshotter) reconcileInProgress(ctx context.Context, requestName string, request *volumes.SnapshotsRequest, status *volumes.SnapshotsStatus) (retErr error) {
	s.logger.Debugf("Reconciling in-progress volume snapshots request %s", requestName)
	if status.Phase != volumes.RequestPhaseInProgress {
		return fmt.Errorf("invalid phase for snapshot request %s, expected %s, got %s", requestName, volumes.RequestPhaseInProgress, status.Phase)
	}
	defer s.logger.Debugf("Reconciled in-progress volume snapshots request %s", requestName)

	continueReconciling := false
	defer func() {
		if retErr == nil {
			return
		}
		status.Phase = volumes.RequestPhaseFailed
		status.Error.Message = retErr.Error()
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
			newStatus, err := s.reconcileInProgressPVC(ctx, requestName, volumeSnapshotRequest, snapshotStatus)
			status.Snapshots[pvcName] = newStatus
			if err != nil {
				return fmt.Errorf("volumes snapshot request %s failed for PVC %s: %w", requestName, pvcName, err)
			}
			if newStatus.Phase == volumes.RequestPhaseInProgress {
				// at least one volume snapshot creation is still in progress
				continueReconciling = true
				continue
			}
		case volumes.RequestPhaseCompleted:
			s.logger.Debugf("VolumeSnapshot for PVC %s has been created successfully", pvcName)
		case volumes.RequestPhaseFailed:
			return fmt.Errorf(
				"volumes snapshot request %s has already failed for PVC %s, previous error: %v",
				requestName,
				pvcName,
				snapshotStatus.Error.Message)
		default:
			return fmt.Errorf("invalid snapshot request phase %s for for PVC %s in volume snapshot request %s", snapshotStatus.Phase, pvcName, requestName)
		}
	}

	if !continueReconciling {
		status.Phase = volumes.RequestPhaseCompleted
	}
	return nil
}

func (s *VolumeSnapshotter) reconcileInProgressPVC(ctx context.Context, requestName string, volumeSnapshotRequest volumes.SnapshotRequest, volumeSnapshotStatus volumes.SnapshotStatus) (status volumes.SnapshotStatus, retErr error) {
	if volumeSnapshotStatus.Phase != volumes.RequestPhaseInProgress {
		return status, fmt.Errorf("invalid volume snapshot request phase %s, expected %s, got %s", requestName, volumes.RequestPhaseInProgress, volumeSnapshotStatus.Phase)
	}
	status = volumeSnapshotStatus
	defer func() {
		if retErr == nil {
			return
		}
		status.Phase = volumes.RequestPhaseFailed
		status.Error.Message = retErr.Error()
	}()

	volumeSnapshotName := fmt.Sprintf("%s-%s", volumeSnapshotRequest.PersistentVolumeClaim.Name, requestName)

	// Check if VolumeSnapshot has been created
	pvcName := types.NamespacedName{
		Namespace: volumeSnapshotRequest.PersistentVolumeClaim.Namespace,
		Name:      volumeSnapshotRequest.PersistentVolumeClaim.Name,
	}
	volumeSnapshot, err := s.snapshotsClient.SnapshotV1().VolumeSnapshots(pvcName.Namespace).Get(ctx, volumeSnapshotName, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		// create new VolumeSnapshot
		_, err = s.createVolumeSnapshotResource(ctx, requestName, volumeSnapshotName, pvcName, volumeSnapshotRequest.VolumeSnapshotClassName)
		if err != nil {
			return status, fmt.Errorf("failed to create VolumeSnapshot for the PersistentVolumeClaim %s: %w", pvcName, err)
		}
		// snapshot creation will take a while, return and check back later in a new reconciliation loop
		return status, nil
	} else if err != nil {
		return status, fmt.Errorf("failed to get VolumeSnapshot %s/%s: %w", volumeSnapshot.Namespace, volumeSnapshot.Name, err)
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

		return status, errors.New(errorMessage)
	}

	// check if VolumeSnapshot is ready
	if volumeSnapshot.Status.ReadyToUse == nil || !*volumeSnapshot.Status.ReadyToUse {
		// VolumeSnapshot is still not ready
		return status, nil
	}

	// VolumeSnapshot is ready -> get VolumeSnapshotContents
	volumeSnapshotContentName := volumeSnapshot.Status.BoundVolumeSnapshotContentName
	if volumeSnapshotContentName == nil || *volumeSnapshotContentName == "" {
		return status, fmt.Errorf("VolumeSnapshot %s/%s does not have bound VolumeSnapshotContent name set", volumeSnapshot.Namespace, volumeSnapshot.Name)
	}
	volumeSnapshotContent, err := s.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Get(ctx, *volumeSnapshotContentName, metav1.GetOptions{})
	if err != nil {
		return status, fmt.Errorf("could not get bound VolumeSnapshotContent '%s' for VolumeSnapshot '%s': %w", *volumeSnapshotContentName, volumeSnapshot.Name, err)
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
		return status, errors.New(errorMessage)
	}

	// check if VolumeSnapshotContent is ready
	if volumeSnapshotContent.Status.ReadyToUse == nil || !*volumeSnapshotContent.Status.ReadyToUse {
		// VolumeSnapshotContent is still not ready
		return status, nil
	}

	// VolumeSnapshotContent is ready -> read the snapshot handle
	if volumeSnapshotContent.Status.SnapshotHandle == nil {
		return status, fmt.Errorf("VolumeSnapshotContent %s (for PersistentVolumeClaim %s) does not have status.snapshotHandle set", volumeSnapshotContent.Name, pvcName.String())
	}
	status.SnapshotHandle = *volumeSnapshotContent.Status.SnapshotHandle
	status.Phase = volumes.RequestPhaseCompleted
	return status, nil
}

func (s *VolumeSnapshotter) createVolumeSnapshotResource(ctx context.Context, requestName, volumeSnapshotName string, pvcName types.NamespacedName, volumeSnapshotClassName string) (*snapshotsv1api.VolumeSnapshot, error) {
	s.logger.Debugf("Create VolumeSnapshot %s for PersistentVolumeClaim %s for snapshot request %s", volumeSnapshotName, pvcName.String(), requestName)

	volumeSnapshot := &snapshotsv1api.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pvcName.Namespace,
			Name:      volumeSnapshotName,
			Labels: map[string]string{
				meta.RequestLabel:              requestName,
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
