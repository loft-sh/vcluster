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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"

	"github.com/loft-sh/vcluster/pkg/config"
)

type VolumeRestorer struct {
	vConfig         *config.VirtualClusterConfig
	kubeClient      *kubernetes.Clientset
	snapshotsClient *snapshotsv1.Clientset
	logger          log.Logger
}

func NewVolumeRestorer(_ context.Context, vConfig *config.VirtualClusterConfig, kubeClient *kubernetes.Clientset, snapshotsClient *snapshotsv1.Clientset, logger log.Logger) (*VolumeRestorer, error) {
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
		vConfig:         vConfig,
		kubeClient:      kubeClient,
		snapshotsClient: snapshotsClient,
		logger:          logger,
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

func (r *VolumeRestorer) restoreVolume(ctx context.Context, volumeSnapshot *snapshotsv1api.VolumeSnapshot) error {
	r.logger.Debugf("Restore volume from VolumeSnapshot %s/%s", volumeSnapshot.Namespace, volumeSnapshot.Name)
	originalPVCJSON, ok := volumeSnapshot.Annotations[persistentVolumeClaimNameAnnotation]
	if !ok {
		return fmt.Errorf("VolumeSnapshot %s/%s does not have a PersistentVolumeClaim JSON annotation set", volumeSnapshot.Namespace, volumeSnapshot.Name)
	}
	var originalPersistentVolumeClaim corev1.PersistentVolumeClaim
	err := json.Unmarshal([]byte(originalPVCJSON), &originalPersistentVolumeClaim)
	if err != nil {
		return fmt.Errorf("failed to unmarshal original PersistentVolumeClaim from JSON: %w", err)
	}

	// 1. Delete the original PersistentVolumeClaim
	err = r.deleteOldPersistentVolumeClaim(ctx, originalPersistentVolumeClaim.Namespace, originalPersistentVolumeClaim.Name)
	if err != nil {
		return fmt.Errorf("failed to delete old PersistentVolumeClaim %s/%s: %w", originalPersistentVolumeClaim.Namespace, originalPersistentVolumeClaim.Name, err)
	}

	// 2. Re-create PersistentVolumeClaim from VolumeSnapshot
	_, err = r.createPersistentVolumeClaimFromSnapshot(ctx, volumeSnapshot, &originalPersistentVolumeClaim)
	if err != nil {
		return fmt.Errorf(
			"failed to create new PersistentVolumeClaim %s/%s from VolumeSnapshot %s/%s: %w",
			originalPersistentVolumeClaim.Namespace,
			originalPersistentVolumeClaim.Name,
			volumeSnapshot.Namespace,
			volumeSnapshot.Name,
			err)
	}

	// 3. Delete VolumeSnapshot resource
	err = r.snapshotsClient.SnapshotV1().VolumeSnapshots(volumeSnapshot.Namespace).Delete(ctx, volumeSnapshot.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete VolumeSnapshot %s/%s: %w", volumeSnapshot.Namespace, volumeSnapshot.Name, err)
	}

	r.logger.Debugf("Restored volume from VolumeSnapshot %s/%s", volumeSnapshot.Namespace, volumeSnapshot.Name)
	return nil
}

func (r *VolumeRestorer) deleteOldPersistentVolumeClaim(ctx context.Context, pvcNamespace, pvcName string) error {
	r.logger.Debugf("Delete original PersistentVolumeClaim %s/%s", pvcNamespace, pvcName)

	// TODO: check if we want to delete the PersistentVolume here.
	// If the PersistentVolume's spec.persistentVolumeReclaimPolicy is 'Delete', the PersistentVolume
	// will be automatically deleted.
	// However, if the PersistentVolume's spec.persistentVolumeReclaimPolicy is 'Retain', the
	// PersistentVolume will not be deleted.

	err := r.kubeClient.CoreV1().PersistentVolumeClaims(pvcNamespace).Delete(ctx, pvcName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete original PersistentVolumeClaim %s/%s: %w", pvcNamespace, pvcName, err)
	}

	err = r.waitForDeleted(ctx, pvcNamespace, pvcName)
	if err != nil {
		return fmt.Errorf("failed to delete original PersistentVolumeClaim %s/%s: %w", pvcNamespace, pvcName, err)
	}

	r.logger.Debugf("Deleted original PersistentVolumeClaim %s/%s", pvcNamespace, pvcName)
	return nil
}

func (r *VolumeRestorer) createPersistentVolumeClaimFromSnapshot(ctx context.Context, volumeSnapshot *snapshotsv1api.VolumeSnapshot, originalPersistentVolumeClaim *corev1.PersistentVolumeClaim) (*corev1.PersistentVolumeClaim, error) {
	r.logger.Debugf(
		"Create new PersistentVolumeClaim %s/%s from VolumeSnapshot %s/%s",
		originalPersistentVolumeClaim.Namespace, originalPersistentVolumeClaim.Name,
		volumeSnapshot.Namespace, volumeSnapshot.Name)

	if originalPersistentVolumeClaim.Namespace != volumeSnapshot.Namespace {
		return nil, errors.New("PersistentVolumeClaim must be restored from a VolumeSnapshot in the same namespace")
	}

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

	r.logger.Debugf(
		"Created new PersistentVolumeClaim %s/%s from VolumeSnapshot %s/%s",
		originalPersistentVolumeClaim.Namespace, originalPersistentVolumeClaim.Name,
		volumeSnapshot.Namespace, volumeSnapshot.Name)

	return restoredPersistentVolumeClaim, nil
}
