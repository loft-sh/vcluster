package csi

import (
	"context"
	"encoding/json"
	"fmt"

	snapshotsv1api "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	snapshotsv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned"
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

// createPreProvisionedVolumeSnapshot creates the pre-provisioned VolumeSnapshot
func (h *snapshotHandler) createPreProvisionedVolumeSnapshot(ctx context.Context, requestLabel, requestName string, volumeSnapshotRequest volumes.SnapshotRequest) (*snapshotsv1api.VolumeSnapshot, error) {
	volumeSnapshotName := fmt.Sprintf("%s-%s", volumeSnapshotRequest.PersistentVolumeClaim.Name, requestName)
	h.logger.Debugf(
		"Create VolumeSnapshot %s for PersistentVolumeClaim %s/%s for request %s",
		volumeSnapshotName,
		volumeSnapshotRequest.PersistentVolumeClaim.Namespace,
		volumeSnapshotRequest.PersistentVolumeClaim.Name,
		requestName)

	volumeSnapshot := &snapshotsv1api.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: volumeSnapshotRequest.PersistentVolumeClaim.Namespace,
			Name:      volumeSnapshotName,
			Labels: map[string]string{
				requestLabel:                   requestName,
				persistentVolumeClaimNameLabel: volumeSnapshotRequest.PersistentVolumeClaim.Name,
			},
		},
		Spec: snapshotsv1api.VolumeSnapshotSpec{
			Source: snapshotsv1api.VolumeSnapshotSource{
				VolumeSnapshotContentName: ptr.To(volumeSnapshotName),
			},
		},
	}
	if volumeSnapshotRequest.VolumeSnapshotClassName != "" {
		volumeSnapshot.Spec.VolumeSnapshotClassName = &volumeSnapshotRequest.VolumeSnapshotClassName
	}

	var err error
	volumeSnapshot, err = h.snapshotsClient.SnapshotV1().VolumeSnapshots(volumeSnapshotRequest.PersistentVolumeClaim.Namespace).Create(ctx, volumeSnapshot, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf(
			"could not create VolumeSnapshot resource for the PersistentVolumeClaim %s/%s: %w",
			volumeSnapshotRequest.PersistentVolumeClaim.Namespace,
			volumeSnapshotRequest.PersistentVolumeClaim.Name,
			err)
	}
	h.logger.Infof(
		"Created VolumeSnapshot resource %s/%s for the PersistentVolumeClaim %s/%s",
		volumeSnapshot.Namespace, volumeSnapshot.Name,
		volumeSnapshotRequest.PersistentVolumeClaim.Namespace,
		volumeSnapshotRequest.PersistentVolumeClaim.Name)

	return volumeSnapshot, nil
}

// createPreProvisionedVolumeSnapshot creates the pre-provisioned VolumeSnapshotContent
func (h *snapshotHandler) createVolumeSnapshotContentResource(
	ctx context.Context,
	requestLabel,
	requestName string,
	snapshotRequest volumes.SnapshotRequest,
	snapshotHandle string,
	deletionPolicy snapshotsv1api.DeletionPolicy) (*snapshotsv1api.VolumeSnapshotContent, error) {
	volumeSnapshotContentName := fmt.Sprintf("%s-%s", snapshotRequest.PersistentVolumeClaim.Name, requestName)
	h.logger.Debugf(
		"Create VolumeSnapshotContent %s for PersistentVolumeClaim %s/%s for request %s",
		volumeSnapshotContentName,
		snapshotRequest.PersistentVolumeClaim.Namespace,
		snapshotRequest.PersistentVolumeClaim.Name,
		requestName)

	volumeSnapshotContent := &snapshotsv1api.VolumeSnapshotContent{
		ObjectMeta: metav1.ObjectMeta{
			Name: volumeSnapshotContentName,
			Labels: map[string]string{
				requestLabel:                   requestName,
				persistentVolumeClaimNameLabel: snapshotRequest.PersistentVolumeClaim.Name,
			},
		},
		Spec: snapshotsv1api.VolumeSnapshotContentSpec{
			DeletionPolicy: deletionPolicy,
			Driver:         snapshotRequest.CSIDriver,
			Source: snapshotsv1api.VolumeSnapshotContentSource{
				SnapshotHandle: &snapshotHandle,
			},
			VolumeSnapshotRef: corev1.ObjectReference{
				Name:      volumeSnapshotContentName,
				Namespace: snapshotRequest.PersistentVolumeClaim.Namespace,
			},
			SourceVolumeMode: snapshotRequest.PersistentVolumeClaim.Spec.VolumeMode,
		},
	}
	if snapshotRequest.VolumeSnapshotClassName != "" {
		volumeSnapshotContent.Spec.VolumeSnapshotClassName = &snapshotRequest.VolumeSnapshotClassName
	}

	var err error
	volumeSnapshotContent, err = h.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Create(ctx, volumeSnapshotContent, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf(
			"could not create VolumeSnapshotContent resource for the PersistentVolumeClaim %s/%s: %w",
			snapshotRequest.PersistentVolumeClaim.Namespace,
			snapshotRequest.PersistentVolumeClaim.Name,
			err)
	}
	h.logger.Infof("Created VolumeSnapshotContent resource %s for the PersistentVolumeClaim %s/%s",
		volumeSnapshotContent.Name,
		snapshotRequest.PersistentVolumeClaim.Namespace,
		snapshotRequest.PersistentVolumeClaim.Name)

	return volumeSnapshotContent, nil
}

// deleteVolumeSnapshot deletes the VolumeSnapshot and the VolumeSnapshotContent with the deletion policy set
// to Delete, so it deletes the VolumeSnapshot and the VolumeSnapshotContent resources, as well as the volume snapshot
// from the storage backend.
func (h *snapshotHandler) deleteVolumeSnapshot(ctx context.Context, requestLabel, requestName string, volumeSnapshotRequest volumes.SnapshotRequest, snapshotHandle string, recreateResourceIfNotFound bool) (bool, error) {
	volumeSnapshotNamespace := volumeSnapshotRequest.PersistentVolumeClaim.Namespace
	volumeSnapshotName := fmt.Sprintf("%s-%s", volumeSnapshotRequest.PersistentVolumeClaim.Name, requestName)
	var volumeSnapshotContentName string
	if recreateResourceIfNotFound {
		volumeSnapshotContentName = volumeSnapshotName
	}
	volumeSnapshot, volumeSnapshotContent, err := h.getVolumeSnapshotResources(ctx, volumeSnapshotNamespace, volumeSnapshotName, volumeSnapshotContentName)
	if err != nil {
		return false, fmt.Errorf("failed to get volume snapshot resources for VolumeSnapshot %s/%s: %w", volumeSnapshotNamespace, volumeSnapshotName, err)
	}

	resourceRecreated := false
	if volumeSnapshotContent == nil && recreateResourceIfNotFound {
		h.logger.Debugf("VolumeSnapshotContent %s not found, recreate it", volumeSnapshotContentName)
		_, err = h.createVolumeSnapshotContentResource(ctx, requestLabel, requestName, volumeSnapshotRequest, snapshotHandle, snapshotsv1api.VolumeSnapshotContentDelete)
		if err != nil {
			return false, fmt.Errorf("failed to recreate VolumeSnapshotContent %s: %w", volumeSnapshotContentName, err)
		}
		resourceRecreated = true
	}
	if volumeSnapshot == nil && recreateResourceIfNotFound {
		h.logger.Debugf("VolumeSnapshot %s/%s not found, recreate it", volumeSnapshotNamespace, volumeSnapshotName)
		_, err = h.createPreProvisionedVolumeSnapshot(ctx, requestLabel, requestName, volumeSnapshotRequest)
		if err != nil {
			return false, fmt.Errorf("failed to recreate VolumeSnapshot %s/%s: %w", volumeSnapshotNamespace, volumeSnapshotName, err)
		}
		resourceRecreated = true
	}
	if resourceRecreated {
		return false, nil
	}

	if volumeSnapshot == nil && volumeSnapshotContent == nil {
		// both the VolumeSnapshot and the VolumeSnapshotContent have been deleted
		return true, nil
	}
	if volumeSnapshot != nil {
		volumeSnapshotJSON, _ := json.Marshal(volumeSnapshot)
		h.logger.Debugf("VolumeSnapshot %s/%s still not deleted: %s", volumeSnapshot.Namespace, volumeSnapshot.Name, volumeSnapshotJSON)
	}
	if volumeSnapshotContent != nil {
		volumeSnapshotContentJSON, _ := json.Marshal(volumeSnapshotContent)
		h.logger.Debugf("VolumeSnapshotContent %s still not deleted: %s", volumeSnapshotContent.Name, volumeSnapshotContentJSON)
	}

	err = h.updateAndDeleteVolumeSnapshotResource(ctx, volumeSnapshot, volumeSnapshotContent, snapshotsv1api.VolumeSnapshotContentDelete)
	if err != nil {
		return false, fmt.Errorf("failed to delete volume snapshot: %w", err)
	}
	return false, nil
}

// cleanupVolumeSnapshotResource deletes the VolumeSnapshot and the VolumeSnapshotContent with the deletion policy set
// to Retain, so only VolumeSnapshot and VolumeSnapshotContent resources are deleted, and the volume snapshot remains
// saved in the storage backend.
func (h *snapshotHandler) cleanupVolumeSnapshotResource(ctx context.Context, volumeSnapshotNamespace, volumeSnapshotName string) (bool, error) {
	volumeSnapshot, volumeSnapshotContent, err := h.getVolumeSnapshotResources(ctx, volumeSnapshotNamespace, volumeSnapshotName, "")
	if err != nil {
		return false, fmt.Errorf("failed to get volume snapshot resources for VolumeSnapshot %s/%s: %w", volumeSnapshotNamespace, volumeSnapshotName, err)
	}
	if volumeSnapshot == nil && volumeSnapshotContent == nil {
		return true, nil
	}
	err = h.updateAndDeleteVolumeSnapshotResource(ctx, volumeSnapshot, volumeSnapshotContent, snapshotsv1api.VolumeSnapshotContentRetain)
	if err != nil {
		return false, fmt.Errorf("failed to cleanup volume snapshot resources: %w", err)
	}
	return false, nil
}

func (h *snapshotHandler) updateAndDeleteVolumeSnapshotResource(
	ctx context.Context,
	volumeSnapshot *snapshotsv1api.VolumeSnapshot,
	volumeSnapshotContent *snapshotsv1api.VolumeSnapshotContent,
	requiredVolumeSnapshotContentDeletionPolicy snapshotsv1api.DeletionPolicy) error {
	if volumeSnapshotContent != nil &&
		volumeSnapshotContent.DeletionTimestamp.IsZero() &&
		volumeSnapshotContent.Spec.DeletionPolicy != requiredVolumeSnapshotContentDeletionPolicy {
		// Patch VolumeSnapshotContent to set DeletionPolicy to the required value!
		// 1. DeletionPolicy=Retain when cleaning up volume snapshot resources
		// 2. DeletionPolicy=Delete when deleting the volume snapshots
		err := h.setVolumeSnapshotContentDeletionPolicy(ctx, volumeSnapshotContent.Name, requiredVolumeSnapshotContentDeletionPolicy)
		if err != nil {
			return fmt.Errorf("failed to set VolumeSnapshotContent %s DeletionPolicy to %s: %w", volumeSnapshotContent.Name, requiredVolumeSnapshotContentDeletionPolicy, err)
		}
		return nil
	}

	err := h.deleteVolumeSnapshotResources(ctx, volumeSnapshot, volumeSnapshotContent)
	if err != nil {
		return fmt.Errorf("failed to delete VolumeSnapshot and/or VolumeSnapshotContent: %w", err)
	}
	return nil
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
		h.logger.Debugf("Delete VolumeSnapshot %s/%s", volumeSnapshot.Namespace, volumeSnapshot.Name)
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
		h.logger.Debugf("Delete VolumeSnapshotContent %s", volumeSnapshotContent.Name)
		err := h.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Delete(ctx, volumeSnapshotContent.Name, metav1.DeleteOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete VolumeSnapshotContent %s: %w", volumeSnapshotContent.Name, err)
		}
	}
	return nil
}

func (h *snapshotHandler) getVolumeSnapshotResources(
	ctx context.Context,
	volumeSnapshotNamespace,
	volumeSnapshotName,
	volumeSnapshotContentName string) (*snapshotsv1api.VolumeSnapshot, *snapshotsv1api.VolumeSnapshotContent, error) {
	volumeSnapshot, err := h.snapshotsClient.SnapshotV1().VolumeSnapshots(volumeSnapshotNamespace).Get(ctx, volumeSnapshotName, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return nil, nil, fmt.Errorf("failed to get VolumeSnapshot %s/%s: %w", volumeSnapshotNamespace, volumeSnapshotName, err)
	}
	if kerrors.IsNotFound(err) && volumeSnapshotContentName == "" {
		return nil, nil, nil
	}
	if kerrors.IsNotFound(err) {
		// while testing, it looked like the snapshots module sometimes returns a non-nil value,
		// even when the VolumeSnapshot is not found
		volumeSnapshot = nil
	}

	if volumeSnapshotContentName == "" && volumeSnapshot != nil {
		if volumeSnapshot.Spec.Source.PersistentVolumeClaimName != nil &&
			volumeSnapshot.Status != nil &&
			volumeSnapshot.Status.BoundVolumeSnapshotContentName != nil {
			// get the dynamically created VolumeSnapshotContent name
			volumeSnapshotContentName = *volumeSnapshot.Status.BoundVolumeSnapshotContentName
		} else if volumeSnapshot.Spec.Source.VolumeSnapshotContentName != nil {
			// get the pre-provisioned VolumeSnapshotContent name
			volumeSnapshotContentName = *volumeSnapshot.Spec.Source.VolumeSnapshotContentName
		}
	}

	if volumeSnapshotContentName == "" {
		return volumeSnapshot, nil, nil
	}

	volumeSnapshotContent, err := h.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Get(ctx, volumeSnapshotContentName, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return nil, nil, fmt.Errorf("failed to get VolumeSnapshotContent %s: %w", volumeSnapshotContentName, err)
	}
	if kerrors.IsNotFound(err) {
		return volumeSnapshot, nil, nil
	}
	return volumeSnapshot, volumeSnapshotContent, nil
}
