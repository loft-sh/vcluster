package csi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	snapshotsv1api "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	snapshotsv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/config"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
)

type VolumeSnapshotRestoreStatus string

const (
	VolumeSnapshotRestoreStatusLabel                                 = "vcluster.loft.sh/volumesnapshot-restore-status"
	VolumeSnapshotRestoreStatusStarted   VolumeSnapshotRestoreStatus = "started"
	VolumeSnapshotRestoreStatusSkipped   VolumeSnapshotRestoreStatus = "skipped"
	VolumeSnapshotRestoreStatusCompleted VolumeSnapshotRestoreStatus = "completed"
	VolumeSnapshotRestoreStatusError     VolumeSnapshotRestoreStatus = "error"
)

type VolumeRestorer struct {
	snapshotHandler
	vConfig *config.VirtualClusterConfig
}

func NewVolumeRestorer(vConfig *config.VirtualClusterConfig, kubeClient *kubernetes.Clientset, snapshotsClient *snapshotsv1.Clientset, logger log.Logger) (*VolumeRestorer, error) {
	if vConfig == nil {
		return nil, errors.New("virtual cluster config is required")
	}
	if kubeClient == nil {
		return nil, errors.New("kubernetes client is required")
	}
	if snapshotsClient == nil {
		return nil, errors.New("snapshot client is required")
	}
	if logger == nil {
		return nil, errors.New("logger is required")
	}

	restorer := &VolumeRestorer{
		snapshotHandler: snapshotHandler{
			kubeClient:      kubeClient,
			snapshotsClient: snapshotsClient,
			logger:          logger,
		},
		vConfig: vConfig,
	}
	return restorer, nil
}

func (r *VolumeRestorer) RestoreVolumes(ctx context.Context, volumeSnapshots []snapshotsv1api.VolumeSnapshot) error {
	r.logger.Info("Start restoring volumes from CSI VolumeSnapshots")
	defer r.logger.Info("Finished restoring volumes from CSI VolumeSnapshots")

	var wg sync.WaitGroup
	maxVolumes := len(volumeSnapshots)
	errCh := make(chan error, maxVolumes)

	for _, volumeSnapshot := range volumeSnapshots {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := r.restoreVolume(ctx, &volumeSnapshot)
			if err != nil {
				errCh <- err
			}
		}()
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	// aggregate all the errors
	var allErrors []error
	for err := range errCh {
		allErrors = append(allErrors, err)
	}

	return errors.Join(allErrors...)
}

func (r *VolumeRestorer) restoreVolume(ctx context.Context, volumeSnapshot *snapshotsv1api.VolumeSnapshot) (retErr error) {
	volumeSnapshotNamespacedName := types.NamespacedName{
		Namespace: volumeSnapshot.Namespace,
		Name:      volumeSnapshot.Name,
	}
	if volumeSnapshot.Spec.Source.VolumeSnapshotContentName == nil {
		return fmt.Errorf("VolumeSnapshot %s does not have 'spec.source.volumeSnapshotContentName' set", volumeSnapshotNamespacedName)
	}
	volumeSnapshotContentName := *volumeSnapshot.Spec.Source.VolumeSnapshotContentName

	r.logger.Infof("Restore volume from VolumeSnapshot %s", volumeSnapshotNamespacedName)
	defer func() {
		if retErr != nil {
			_, err := r.setVolumeSnapshotRestoreStatus(ctx, volumeSnapshotNamespacedName, volumeSnapshotContentName, VolumeSnapshotRestoreStatusError)
			if err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("failed to set VolumeSnapshot restore status error: %v", err))
			}
		}
	}()

	volumeSnapshotStatus, ok := volumeSnapshot.Labels[VolumeSnapshotRestoreStatusLabel]
	if ok && VolumeSnapshotRestoreStatus(volumeSnapshotStatus) == VolumeSnapshotRestoreStatusCompleted {
		r.logger.Infof("Skip restoring volume from VolumeSnapshot %s because the restore has been already completed", volumeSnapshotNamespacedName)
		return nil
	}

	// initial wait
	//r.logger.Info("WAIT 30s before VolumeSnapshot update")
	//time.Sleep(30 * time.Second)

	// Set VolumeSnapshot restore status to 'started'
	var err error
	volumeSnapshot, err = r.setVolumeSnapshotRestoreStatus(ctx, volumeSnapshotNamespacedName, volumeSnapshotContentName, VolumeSnapshotRestoreStatusStarted)
	if err != nil {
		return fmt.Errorf("failed to update VolumeSnapshot %s restore status %w", volumeSnapshotNamespacedName, err)
	}

	// wait after update
	//r.logger.Info("WAIT 30s after VolumeSnapshot update")
	//time.Sleep(30 * time.Second)

	// Wait for the VolumeSnapshot to be ready for use
	volumeSnapshot, _, err = r.waitForReadyToUse(ctx, volumeSnapshot.Namespace, volumeSnapshot.Name)
	if err != nil {
		return fmt.Errorf("failed while waiting VolumeSnapshot %s to be for ready to use: %w", volumeSnapshotNamespacedName, err)
	}

	// Get the original PersistentVolumeClaim
	originalPVCJSON, ok := volumeSnapshot.Annotations[persistentVolumeClaimNameAnnotation]
	if !ok {
		return fmt.Errorf("VolumeSnapshot %s/%s does not have a PersistentVolumeClaim JSON annotation set", volumeSnapshot.Namespace, volumeSnapshot.Name)
	}
	var originalPersistentVolumeClaim corev1.PersistentVolumeClaim
	err = json.Unmarshal([]byte(originalPVCJSON), &originalPersistentVolumeClaim)
	if err != nil {
		return fmt.Errorf("failed to unmarshal original PersistentVolumeClaim from JSON: %w", err)
	}

	// Check if the PersistentVolumeClaim already exists
	pvcExists, err := r.checkExistingPersistentVolumeClaim(ctx, originalPersistentVolumeClaim.Namespace, originalPersistentVolumeClaim.Name)
	if err != nil {
		return fmt.Errorf("failed to check if PersistentVolumeClaim %s/%s already exists: %w", originalPersistentVolumeClaim.Namespace, originalPersistentVolumeClaim.Name, err)
	}
	if pvcExists {
		// Set VolumeSnapshot restore status to 'skipped'
		volumeSnapshot, err = r.setVolumeSnapshotRestoreStatus(ctx, volumeSnapshotNamespacedName, volumeSnapshotContentName, VolumeSnapshotRestoreStatusSkipped)
		if err != nil {
			return fmt.Errorf("failed to update VolumeSnapshot %s restore status %w", volumeSnapshotNamespacedName, err)
		}
		r.logger.Infof(
			"Skipped restoring PersistentVolumeClaim %s/%s from VolumeSnapshot %s because the PersistentVolumeClaim already exists",
			originalPersistentVolumeClaim.Namespace,
			originalPersistentVolumeClaim.Name,
			volumeSnapshotNamespacedName)
		return nil
	}

	//// x. Delete the original PersistentVolumeClaim
	//err = r.deleteOldPersistentVolumeClaim(ctx, originalPersistentVolumeClaim.Namespace, originalPersistentVolumeClaim.Name)
	//if err != nil {
	//	return fmt.Errorf("failed to delete old PersistentVolumeClaim %s/%s: %w", originalPersistentVolumeClaim.Namespace, originalPersistentVolumeClaim.Name, err)
	//}

	// Re-create PersistentVolumeClaim from VolumeSnapshot
	_, err = r.createPersistentVolumeClaimFromSnapshot(ctx, volumeSnapshot, &originalPersistentVolumeClaim)
	if err != nil {
		return fmt.Errorf(
			"failed to create new PersistentVolumeClaim %s/%s from VolumeSnapshot %s: %w",
			originalPersistentVolumeClaim.Namespace,
			originalPersistentVolumeClaim.Name,
			volumeSnapshotNamespacedName,
			err)
	}

	// Set VolumeSnapshot restore status to 'completed'
	_, err = r.setVolumeSnapshotRestoreStatus(ctx, volumeSnapshotNamespacedName, volumeSnapshotContentName, VolumeSnapshotRestoreStatusCompleted)
	if err != nil {
		return fmt.Errorf("failed to update VolumeSnapshot %s restore status %w", volumeSnapshotNamespacedName, err)
	}

	// TODO: add config so users can select if the VolumeSnapshot is deleted after PVC has been successfully restored

	r.logger.Infof("Restored volume from VolumeSnapshot %s", volumeSnapshotNamespacedName)
	return nil
}

func (r *VolumeRestorer) setVolumeSnapshotRestoreStatus(ctx context.Context, volumeSnapshotNamespacedName types.NamespacedName, volumeSnapshotContentName string, restoreStatus VolumeSnapshotRestoreStatus) (*snapshotsv1api.VolumeSnapshot, error) {
	r.logger.Debugf("Update VolumeSnapshot %s restore status to %s", volumeSnapshotNamespacedName, restoreStatus)

	labelPatch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": map[string]string{
				VolumeSnapshotRestoreStatusLabel: string(restoreStatus),
			},
		},
	}
	labelPatchBytes, err := json.Marshal(labelPatch)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal VolumeSnapshot %s label '%s: %s' patch: %w", volumeSnapshotNamespacedName, VolumeSnapshotRestoreStatusLabel, restoreStatus, err)
	}

	patchedVolumeSnapshot, err := r.snapshotsClient.SnapshotV1().VolumeSnapshots(volumeSnapshotNamespacedName.Namespace).Patch(ctx, volumeSnapshotNamespacedName.Name, types.MergePatchType, labelPatchBytes, metav1.PatchOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to patch VolumeSnapshot %s with label %s: %s: %w", volumeSnapshotNamespacedName, VolumeSnapshotRestoreStatusLabel, restoreStatus, err)
	}
	_, err = r.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Patch(ctx, volumeSnapshotContentName, types.MergePatchType, labelPatchBytes, metav1.PatchOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to patch VolumeSnapshotContent %s with label %s: %s: %w", volumeSnapshotContentName, VolumeSnapshotRestoreStatusLabel, restoreStatus, err)
	}

	r.logger.Debugf("Updated VolumeSnapshot %s restore status to %s", volumeSnapshotNamespacedName, restoreStatus)
	return patchedVolumeSnapshot, nil
}

func (r *VolumeRestorer) checkExistingPersistentVolumeClaim(ctx context.Context, pvcNamespace, pvcName string) (bool, error) {
	r.logger.Infof("Check if PersistentVolumeClaim %s/%s exists", pvcNamespace, pvcName)

	_, err := r.kubeClient.CoreV1().PersistentVolumeClaims(pvcNamespace).Get(ctx, pvcName, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to get PersistentVolumeClaim %s/%s: %w", pvcNamespace, pvcName, err)
	}

	return true, nil
}

func (r *VolumeRestorer) deleteOldPersistentVolumeClaim(ctx context.Context, pvcNamespace, pvcName string) error {
	r.logger.Infof("Delete original PersistentVolumeClaim %s/%s", pvcNamespace, pvcName)

	// TODO: check if we want to delete the PersistentVolume here.
	//  If the PersistentVolume's spec.persistentVolumeReclaimPolicy is 'Delete', the PersistentVolume
	//  will be automatically deleted.
	//  However, if the PersistentVolume's spec.persistentVolumeReclaimPolicy is 'Retain', the
	//  PersistentVolume will not be deleted.

	// TODO: check if PersistentVolumeClaim has finalizers that should be removed! Log (debug) all finalizers
	//  before trying to delete the PVC.

	err := r.kubeClient.CoreV1().PersistentVolumeClaims(pvcNamespace).Delete(ctx, pvcName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete original PersistentVolumeClaim %s/%s: %w", pvcNamespace, pvcName, err)
	}

	err = r.waitForPersistentVolumeClaimDeleted(ctx, pvcNamespace, pvcName)
	if err != nil {
		return fmt.Errorf("failed to delete original PersistentVolumeClaim %s/%s: %w", pvcNamespace, pvcName, err)
	}

	r.logger.Infof("Deleted original PersistentVolumeClaim %s/%s", pvcNamespace, pvcName)
	return nil
}

func (r *VolumeRestorer) createPersistentVolumeClaimFromSnapshot(ctx context.Context, volumeSnapshot *snapshotsv1api.VolumeSnapshot, originalPersistentVolumeClaim *corev1.PersistentVolumeClaim) (*corev1.PersistentVolumeClaim, error) {
	r.logger.Infof(
		"Create new PersistentVolumeClaim %s/%s from VolumeSnapshot %s/%s",
		originalPersistentVolumeClaim.Namespace, originalPersistentVolumeClaim.Name,
		volumeSnapshot.Namespace, volumeSnapshot.Name)

	if originalPersistentVolumeClaim.Namespace != volumeSnapshot.Namespace {
		return nil, errors.New("PersistentVolumeClaim must be restored from a VolumeSnapshot in the same namespace")
	}

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
		return nil, fmt.Errorf(
			"failed to create PersistentVolumeClaim %s/%s from VolumeSnapshot %s/%s: %w",
			originalPersistentVolumeClaim.Namespace,
			originalPersistentVolumeClaim.Name,
			volumeSnapshot.Namespace,
			volumeSnapshot.Name,
			err)
	}

	r.logger.Infof(
		"Created new PersistentVolumeClaim %s/%s from VolumeSnapshot %s/%s",
		originalPersistentVolumeClaim.Namespace, originalPersistentVolumeClaim.Name,
		volumeSnapshot.Namespace, volumeSnapshot.Name)

	return restoredPersistentVolumeClaim, nil
}
