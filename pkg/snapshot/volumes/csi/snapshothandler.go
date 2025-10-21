package csi

import (
	"context"
	"encoding/json"
	"fmt"

	snapshotsv1api "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	snapshotsv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
)

type snapshotHandler struct {
	kubeClient      *kubernetes.Clientset
	snapshotsClient *snapshotsv1.Clientset
	eventRecorder   record.EventRecorder
	logger          loghelper.Logger
}

// createVolumeSnapshotResource creates the pre-provisioned VolumeSnapshot
func (h *snapshotHandler) createVolumeSnapshotResource(ctx context.Context, requestName, volumeSnapshotName string, pvcName types.NamespacedName, volumeSnapshotClassName string) (*snapshotsv1api.VolumeSnapshot, error) {
	h.logger.Debugf("Create VolumeSnapshot %s for PersistentVolumeClaim %s for restore request %s", volumeSnapshotName, pvcName.String(), requestName)

	volumeSnapshot := &snapshotsv1api.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pvcName.Namespace,
			Name:      volumeSnapshotName,
			Labels: map[string]string{
				constants.RestoreRequestLabel:  requestName,
				persistentVolumeClaimNameLabel: pvcName.Name,
			},
		},
		Spec: snapshotsv1api.VolumeSnapshotSpec{
			Source: snapshotsv1api.VolumeSnapshotSource{
				VolumeSnapshotContentName: ptr.To(volumeSnapshotName),
			},
		},
	}
	if volumeSnapshotClassName != "" {
		volumeSnapshot.Spec.VolumeSnapshotClassName = &volumeSnapshotClassName
	}

	var err error
	volumeSnapshot, err = h.snapshotsClient.SnapshotV1().VolumeSnapshots(pvcName.Namespace).Create(ctx, volumeSnapshot, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not create VolumeSnapshot resource for the PersistentVolumeClaim %s: %w", pvcName, err)
	}
	h.logger.Infof("Created VolumeSnapshot resource %s/%s for the PersistentVolumeClaim %s", volumeSnapshot.Namespace, volumeSnapshot.Name, pvcName)

	return volumeSnapshot, nil
}

// createVolumeSnapshotResource creates the pre-provisioned VolumeSnapshotContent
func (h *snapshotHandler) createVolumeSnapshotContentResource(ctx context.Context, requestName, volumeSnapshotName string, volumeRestoreRequest volumes.RestoreRequest) (*snapshotsv1api.VolumeSnapshotContent, error) {
	h.logger.Debugf(
		"Create VolumeSnapshotContent %s for PersistentVolumeClaim %s/%s for request %s",
		volumeSnapshotName,
		volumeRestoreRequest.PersistentVolumeClaim.Namespace,
		volumeRestoreRequest.PersistentVolumeClaim.Name,
		requestName)

	volumeSnapshotContent := &snapshotsv1api.VolumeSnapshotContent{
		ObjectMeta: metav1.ObjectMeta{
			Name: volumeSnapshotName,
			Labels: map[string]string{
				constants.RestoreRequestLabel:  requestName,
				persistentVolumeClaimNameLabel: volumeRestoreRequest.PersistentVolumeClaim.Name,
			},
		},
		Spec: snapshotsv1api.VolumeSnapshotContentSpec{
			DeletionPolicy: snapshotsv1api.VolumeSnapshotContentRetain,
			Driver:         volumeRestoreRequest.CSIDriver,
			Source: snapshotsv1api.VolumeSnapshotContentSource{
				SnapshotHandle: ptr.To(volumeRestoreRequest.SnapshotHandle),
			},
			VolumeSnapshotRef: corev1.ObjectReference{
				Name:      volumeSnapshotName,
				Namespace: volumeRestoreRequest.PersistentVolumeClaim.Namespace,
			},
		},
	}
	if volumeRestoreRequest.VolumeSnapshotClassName != "" {
		volumeSnapshotContent.Spec.VolumeSnapshotClassName = &volumeRestoreRequest.VolumeSnapshotClassName
	}
	if volumeRestoreRequest.PersistentVolumeClaim.Spec.VolumeMode != nil {
		volumeSnapshotContent.Spec.SourceVolumeMode = volumeRestoreRequest.PersistentVolumeClaim.Spec.VolumeMode
	}

	var err error
	volumeSnapshotContent, err = h.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Create(ctx, volumeSnapshotContent, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf(
			"could not create VolumeSnapshotContent resource for the PersistentVolumeClaim %s/%s: %w",
			volumeRestoreRequest.PersistentVolumeClaim.Namespace,
			volumeRestoreRequest.PersistentVolumeClaim.Name,
			err)
	}
	h.logger.Infof("Created VolumeSnapshotContent resource %s for the PersistentVolumeClaim %s/%s",
		volumeSnapshotContent.Name,
		volumeRestoreRequest.PersistentVolumeClaim.Namespace,
		volumeRestoreRequest.PersistentVolumeClaim.Name)

	return volumeSnapshotContent, nil
}

// deleteVolumeSnapshot deletes the VolumeSnapshot and the VolumeSnapshotContent with the deletion policy set
// to Delete, so it deletes the VolumeSnapshot and the VolumeSnapshotContent resources, as well as the volume snapshot
// from the storage backend.
func (h *snapshotHandler) deleteVolumeSnapshot(ctx context.Context, volumeSnapshotNamespace, volumeSnapshotName string) (bool, error) {
	deleted, err := h.findAndDeleteVolumeSnapshotResource(ctx, volumeSnapshotNamespace, volumeSnapshotName, snapshotsv1api.VolumeSnapshotContentDelete)
	if err != nil {
		return false, fmt.Errorf("failed to delete volume snapshot: %w", err)
	}
	return deleted, nil
}

// cleanupVolumeSnapshotResource deletes the VolumeSnapshot and the VolumeSnapshotContent with the deletion policy set
// to Retain, so only VolumeSnapshot and VolumeSnapshotContent resources are deleted, and the volume snapshot remains
// saved in the storage backend.
func (h *snapshotHandler) cleanupVolumeSnapshotResource(ctx context.Context, volumeSnapshotNamespace, volumeSnapshotName string) (bool, error) {
	deleted, err := h.findAndDeleteVolumeSnapshotResource(ctx, volumeSnapshotNamespace, volumeSnapshotName, snapshotsv1api.VolumeSnapshotContentRetain)
	if err != nil {
		return false, fmt.Errorf("failed to cleanup volume snapshot resources: %w", err)
	}
	return deleted, nil
}

func (h *snapshotHandler) findAndDeleteVolumeSnapshotResource(
	ctx context.Context,
	volumeSnapshotNamespace,
	volumeSnapshotName string,
	requiredVolumeSnapshotContentDeletionPolicy snapshotsv1api.DeletionPolicy) (bool, error) {
	volumeSnapshot, err := h.snapshotsClient.SnapshotV1().VolumeSnapshots(volumeSnapshotNamespace).Get(ctx, volumeSnapshotName, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return false, fmt.Errorf("failed to get VolumeSnapshot %s/%s: %w", volumeSnapshotNamespace, volumeSnapshotName, err)
	}
	if kerrors.IsNotFound(err) {
		return true, nil
	}

	var volumeSnapshotContentName string
	if volumeSnapshot.Spec.Source.PersistentVolumeClaimName != nil &&
		volumeSnapshot.Status != nil &&
		volumeSnapshot.Status.BoundVolumeSnapshotContentName != nil {
		// get the dynamically created VolumeSnapshotContent name
		volumeSnapshotContentName = *volumeSnapshot.Status.BoundVolumeSnapshotContentName
	} else if volumeSnapshot.Spec.Source.VolumeSnapshotContentName != nil {
		// get the pre-provisioned VolumeSnapshotContent name
		volumeSnapshotContentName = *volumeSnapshot.Spec.Source.VolumeSnapshotContentName
	}

	var volumeSnapshotContent *snapshotsv1api.VolumeSnapshotContent
	if volumeSnapshotContentName != "" {
		volumeSnapshotContent, err = h.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Get(ctx, volumeSnapshotContentName, metav1.GetOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			return false, fmt.Errorf("failed to get VolumeSnapshotContent %s: %w", volumeSnapshotContentName, err)
		}
		if !kerrors.IsNotFound(err) &&
			volumeSnapshotContent != nil &&
			volumeSnapshotContent.DeletionTimestamp.IsZero() &&
			volumeSnapshotContent.Spec.DeletionPolicy != requiredVolumeSnapshotContentDeletionPolicy {
			//
			// Patch VolumeSnapshotContent to set DeletionPolicy to the required value!
			// 1. DeletionPolicy=Retain when cleaning up volume snapshot resources
			// 2. DeletionPolicy=Delete when deleting the volume snapshots
			//
			err = h.setVolumeSnapshotContentDeletionPolicy(ctx, volumeSnapshotContentName, requiredVolumeSnapshotContentDeletionPolicy)
			if err != nil {
				return false, fmt.Errorf("failed to set VolumeSnapshotContent %s DeletionPolicy to Retain: %w", volumeSnapshotContentName, err)
			}
			return false, nil
		}
	}

	err = h.deleteVolumeSnapshotResources(ctx, volumeSnapshot, volumeSnapshotContent)
	if err != nil {
		return false, fmt.Errorf("failed to delete VolumeSnapshot %s/%s and/or VolumeSnapshotContent %s: %w", volumeSnapshotNamespace, volumeSnapshotName, volumeSnapshotContentName, err)
	}
	return true, nil
}

func (h *snapshotHandler) setVolumeSnapshotContentDeletionPolicy(ctx context.Context, volumeSnapshotContentName string, deletionPolicy snapshotsv1api.DeletionPolicy) error {
	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"deletionPolicy": string(deletionPolicy),
		},
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("failed to marshal VolumeSnapshotContent patch: %w", err)
	}
	_, err = h.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Patch(ctx, volumeSnapshotContentName, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to patch VolumeSnapshotContent %s: %w", volumeSnapshotContentName, err)
	}
	return nil
}

func (h *snapshotHandler) deleteVolumeSnapshotResources(
	ctx context.Context,
	volumeSnapshot *snapshotsv1api.VolumeSnapshot,
	volumeSnapshotContent *snapshotsv1api.VolumeSnapshotContent) error {
	if volumeSnapshot != nil &&
		volumeSnapshot.DeletionTimestamp.IsZero() {
		err := h.snapshotsClient.SnapshotV1().VolumeSnapshots(volumeSnapshot.Namespace).Delete(ctx, volumeSnapshot.Name, metav1.DeleteOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete VolumeSnapshot %s/%s: %w", volumeSnapshot.Namespace, volumeSnapshot.Name, err)
		}
	}
	if volumeSnapshotContent != nil &&
		volumeSnapshotContent.DeletionTimestamp.IsZero() &&
		volumeSnapshotContent.Spec.DeletionPolicy == snapshotsv1api.VolumeSnapshotContentRetain {
		// Delete the VolumeSnapshotContent manually in case it has the Retain deletion policy.
		// Otherwise, the VolumeSnapshotContent resource will be deleted automatically by the snapshot-controller.
		// Here we have 2 cases:
		// 1. DeletionPolicy=Retain when cleaning up volume snapshot resources, where only the VolumeSnapshotContent is
		//    deleted, and the volume snapshot remains saved in the storage backend.
		// 2. DeletionPolicy=Delete when deleting the volume snapshots, where both the VolumeSnapshotContent and the
		//    volume snapshot from the storage backend are deleted.
		err := h.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Delete(ctx, volumeSnapshotContent.Name, metav1.DeleteOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete VolumeSnapshotContent %s: %w", volumeSnapshotContent.Name, err)
		}
	}
	return nil
}
