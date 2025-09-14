package csi

import (
	"context"
	"errors"
	"fmt"
	"github.com/loft-sh/vcluster/pkg/snapshot/meta"

	snapshotsv1api "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (s *VolumeSnapshotter) reconcileInProgress(ctx context.Context, snapshotRequestName string, snapshotRequest *volumes.SnapshotRequest) (retErr error) {
	s.logger.Debugf("Reconciling in-progress volume snapshots request %s", snapshotRequestName)
	if snapshotRequest.Status.Phase != volumes.RequestPhaseInProgress {
		return fmt.Errorf("invalid phase for snapshot request %s, expected %s, got %s", snapshotRequestName, volumes.RequestPhaseInProgress, snapshotRequest.Status.Phase)
	}
	defer s.logger.Debugf("Reconciled in-progress volume snapshots request %s", snapshotRequestName)

	inProgress := false
	defer func() {
		if retErr != nil {
			snapshotRequest.Status.Phase = volumes.RequestPhaseFailed
		}
	}()

	for _, snapshotConfig := range snapshotRequest.Spec.VolumeSnapshotConfigs {
		pvcName := types.NamespacedName{
			Namespace: snapshotConfig.PersistentVolumeClaim.Namespace,
			Name:      snapshotConfig.PersistentVolumeClaim.Name,
		}.String()
		if snapshotRequest.Status.Snapshots == nil {
			snapshotRequest.Status.Snapshots = volumes.Snapshots{}
		}
		snapshotStatus, ok := snapshotRequest.Status.Snapshots[pvcName]
		if !ok {
			snapshotStatus = volumes.SnapshotStatus{
				Phase: volumes.RequestPhaseInProgress,
			}
			snapshotRequest.Status.Snapshots[pvcName] = snapshotStatus
		}
		if snapshotStatus.Phase == volumes.RequestPhaseNotStarted {
			snapshotStatus.Phase = volumes.RequestPhaseInProgress
		}

		switch snapshotStatus.Phase {
		case volumes.RequestPhaseInProgress:
			newStatus, err := s.reconcileInProgressPVC(ctx, snapshotRequestName, snapshotConfig)
			if err != nil {
				return fmt.Errorf("failed to reconcile in-progress volumes snapshot request %s for PVC %s: %w", snapshotRequestName, pvcName, err)
			}
			if newStatus.Equals(snapshotStatus) {
				// at least one volume snapshot creation is still in progress
				inProgress = true
				continue
			}
			snapshotRequest.Status.Snapshots[pvcName] = newStatus
		case volumes.RequestPhaseCompleted:
			s.logger.Debugf("VolumeSnapshot for PVC %s has been created", pvcName)
			// TODO: delete VolumeSnapshot and VolumeSnapshotContent (make sure to use VolumeSnapshotClass with deletion policy 'Retain'
		case volumes.RequestPhaseFailed:
			s.logger.Debugf("Failed to create VolumeSnapshot for PVC %s", pvcName)
			// TODO: cleanup VolumeSnapshot and VolumeSnapshotContent
		default:
			return fmt.Errorf("invalid snapshot request phase %s for for PVC %s in volume snapshot request %s", snapshotStatus.Phase, pvcName, snapshotRequestName)
		}
	}

	if !inProgress {
		snapshotRequest.Status.Phase = volumes.RequestPhaseCompleted
	}
	return nil
}

func (s *VolumeSnapshotter) reconcileInProgressPVC(ctx context.Context, snapshotRequestName string, config volumes.SnapshotConfig) (status volumes.SnapshotStatus, retErr error) {
	status = volumes.SnapshotStatus{
		Phase: volumes.RequestPhaseInProgress,
	}
	defer func() {
		if retErr != nil {
			status.Phase = volumes.RequestPhaseFailed
		}
	}()

	volumeSnapshotName := fmt.Sprintf("%s-%s", config.PersistentVolumeClaim.Name, snapshotRequestName)

	// Check if VolumeSnapshot has been created
	pvcName := types.NamespacedName{
		Namespace: config.PersistentVolumeClaim.Namespace,
		Name:      config.PersistentVolumeClaim.Name,
	}
	volumeSnapshot, err := s.snapshotsClient.SnapshotV1().VolumeSnapshots(pvcName.Namespace).Get(ctx, volumeSnapshotName, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		// create new VolumeSnapshot
		volumeSnapshot, err = s.createVolumeSnapshotResource(ctx, snapshotRequestName, volumeSnapshotName, pvcName, config.VolumeSnapshotClassName)
		if err != nil {
			return volumes.SnapshotStatus{}, fmt.Errorf("failed to create VolumeSnapshot for the PersistentVolumeClaim %s: %w", pvcName, err)
		}
		// snapshot creation will take a while, return and check back later in a new reconciliation loop
		return status, nil
	} else if err != nil {
		return volumes.SnapshotStatus{}, fmt.Errorf("failed to get VolumeSnapshot %s/%s: %w", volumeSnapshot.Namespace, volumeSnapshot.Name, err)
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

func (s *VolumeSnapshotter) createVolumeSnapshotResource(ctx context.Context, snapshotRequestName, volumeSnapshotName string, pvcName types.NamespacedName, volumeSnapshotClassName string) (*snapshotsv1api.VolumeSnapshot, error) {
	s.logger.Debugf("Create VolumeSnapshot %s for PersistentVolumeClaim %s for snapshot request %s", volumeSnapshotName, pvcName.String(), snapshotRequestName)

	volumeSnapshot := &snapshotsv1api.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pvcName.Namespace,
			Name:      volumeSnapshotName,
			Labels: map[string]string{
				meta.RequestLabel:              snapshotRequestName,
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
