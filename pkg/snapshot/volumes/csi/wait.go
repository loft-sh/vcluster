package csi

import (
	"context"
	"fmt"
	"time"

	snapshotsv1api "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	snapshotsv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type snapshotHandler struct {
	kubeClient      *kubernetes.Clientset
	snapshotsClient *snapshotsv1.Clientset
	logger          loghelper.Logger
}

func (s *snapshotHandler) waitForReadyToUse(ctx context.Context, volumeSnapshotNamespace, volumeSnapshotName string) (*snapshotsv1api.VolumeSnapshot, *snapshotsv1api.VolumeSnapshotContent, error) {
	var err error
	var volumeSnapshot *snapshotsv1api.VolumeSnapshot
	var volumeSnapshotContent *snapshotsv1api.VolumeSnapshotContent

	err = wait.PollUntilContextTimeout(ctx, time.Second*5, 15*time.Minute, true, func(ctx context.Context) (bool, error) {
		volumeSnapshot, err = s.snapshotsClient.SnapshotV1().VolumeSnapshots(volumeSnapshotNamespace).Get(ctx, volumeSnapshotName, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("could not get VolumeSnapshot %s: %w", volumeSnapshot.Name, err)
		}

		if volumeSnapshot.Status == nil {
			return false, nil
		}

		if volumeSnapshot.Status.ReadyToUse != nil && *volumeSnapshot.Status.ReadyToUse {
			// get VolumeSnapshotContent resource and check it as well
			boundVolumeSnapshotContentName := volumeSnapshot.Status.BoundVolumeSnapshotContentName
			if boundVolumeSnapshotContentName == nil || *boundVolumeSnapshotContentName == "" {
				return false, fmt.Errorf("VolumeSnapshot %s does not have bound VolumeSnapshotContent name set", volumeSnapshotName)
			}

			// get VolumeSnapshotContent
			volumeSnapshotContent, err = s.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Get(ctx, *boundVolumeSnapshotContentName, metav1.GetOptions{})
			if err != nil {
				return false, fmt.Errorf("could not get bound VolumeSnapshotContent '%s' for VolumeSnapshot '%s': %w", *boundVolumeSnapshotContentName, volumeSnapshotName, err)
			}
			if volumeSnapshotContent.Status.ReadyToUse == nil || !*volumeSnapshotContent.Status.ReadyToUse {
				return false, nil
			}
			if volumeSnapshotContent.Status.SnapshotHandle == nil {
				return false, fmt.Errorf("VolumeSnapshotContent '%s' does not have status.snapshotHandle set", volumeSnapshotContent.Name)
			}

			// VolumeSnapshot is created and ready to use!
			// VolumeSnapshotContent is created, ready to use and has a snapshot handle set!
			return true, nil
		}

		if volumeSnapshot.Status.Error != nil {
			var errorMessage string
			if volumeSnapshot.Status.Error.Message != nil {
				errorMessage = *volumeSnapshot.Status.Error.Message
			}
			return false, fmt.Errorf("VolumeSnapshot %s failed with message '%s'", volumeSnapshot.Name, errorMessage)
		}

		// not ready, no error
		return false, nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("error waiting for VolumeSnapshot %s/%s to be ready: %w", volumeSnapshotNamespace, volumeSnapshotName, err)
	}
	return volumeSnapshot, volumeSnapshotContent, nil
}

func (s *snapshotHandler) waitForVolumeSnapshotDeleted(ctx context.Context, volumeSnapshotNamespace, volumeSnapshotName, volumeSnapshotContentName string) error {
	s.logger.Debugf(
		"Wait until the VolumeSnapshot %s/%s has been deleted",
		volumeSnapshotNamespace,
		volumeSnapshotName)

	err := wait.PollUntilContextTimeout(ctx, time.Second*5, 15*time.Minute, true, func(ctx context.Context) (bool, error) {
		_, err := s.snapshotsClient.SnapshotV1().VolumeSnapshots(volumeSnapshotNamespace).Get(ctx, volumeSnapshotName, metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			return true, nil
		} else if err != nil {
			return false, fmt.Errorf("failed to get VolumeSnapshot %s/%s: %w", volumeSnapshotNamespace, volumeSnapshotName, err)
		}
		return false, nil
	})

	if err != nil {
		return fmt.Errorf("error waiting for VolumeSnapshot %s/%s to be deleted: %w", volumeSnapshotNamespace, volumeSnapshotName, err)
	}

	err = wait.PollUntilContextTimeout(ctx, time.Second*5, 15*time.Minute, true, func(ctx context.Context) (bool, error) {
		_, err := s.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Get(ctx, volumeSnapshotContentName, metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			return true, nil
		} else if err != nil {
			return false, fmt.Errorf("failed to get VolumeSnapshotContent %s/%s: %w", volumeSnapshotNamespace, volumeSnapshotName, err)
		}
		return false, nil
	})

	if err != nil {
		return fmt.Errorf("error waiting for VolumeSnapshotContent %s to be deleted: %w", volumeSnapshotContentName, err)
	}

	s.logger.Debugf(
		"PersistentVolumeClaim %s/%s has been successfully deleted",
		volumeSnapshotNamespace,
		volumeSnapshotName)
	return nil
}

func (r *snapshotHandler) waitForPersistentVolumeClaimDeleted(ctx context.Context, persistentVolumeClaimNamespace, persistentVolumeClaimName string) error {
	r.logger.Debugf(
		"Wait until the PersistentVolumeClaim %s/%s has been deleted",
		persistentVolumeClaimNamespace,
		persistentVolumeClaimName)

	err := wait.PollUntilContextTimeout(ctx, time.Second*5, 15*time.Minute, true, func(ctx context.Context) (bool, error) {
		_, err := r.kubeClient.CoreV1().PersistentVolumeClaims(persistentVolumeClaimNamespace).Get(ctx, persistentVolumeClaimName, metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			return true, nil
		} else if err != nil {
			return false, fmt.Errorf("failed to get PersistentVolumeClaim %s/%s: %w", persistentVolumeClaimNamespace, persistentVolumeClaimName, err)
		}
		return false, nil
	})

	if err != nil {
		return fmt.Errorf("error waiting for PersistentVolumeClaim %s/%s to be deleted: %w", persistentVolumeClaimNamespace, persistentVolumeClaimName, err)
	}

	r.logger.Debugf(
		"PersistentVolumeClaim %s/%s has been successfully deleted",
		persistentVolumeClaimNamespace,
		persistentVolumeClaimName)
	return nil
}
