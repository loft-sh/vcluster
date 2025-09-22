package csi

import (
	"context"
	"errors"
	"fmt"

	snapshotsv1api "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	snapshotsv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/snapshot/meta"
	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
)

type Restorer struct {
	snapshotHandler
	vConfig *config.VirtualClusterConfig
}

func NewRestorer(vConfig *config.VirtualClusterConfig, kubeClient *kubernetes.Clientset, snapshotsClient *snapshotsv1.Clientset, logger loghelper.Logger) (*Restorer, error) {
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

	restorer := &Restorer{
		snapshotHandler: snapshotHandler{
			kubeClient:      kubeClient,
			snapshotsClient: snapshotsClient,
			logger:          logger,
		},
		vConfig: vConfig,
	}
	return restorer, nil
}

// Reconcile volumes restore request.
func (r *Restorer) Reconcile(ctx context.Context, restoreRequestName string, restoreRequest *volumes.SnapshotRequest) error {
	r.logger.Infof("Restore volumes for restore request %s", restoreRequestName)
	var err error

	switch restoreRequest.Status.Phase {
	case volumes.RequestPhaseNotStarted:
		restoreRequest.Status.Phase = volumes.RequestPhaseInProgress
		fallthrough
	case volumes.RequestPhaseInProgress:
		err = r.reconcileInProgress(ctx, restoreRequestName, restoreRequest)
		if err != nil {
			return fmt.Errorf("failed to reconcile failed volumes snapshot request %s: %w", restoreRequestName, err)
		}
	case volumes.RequestPhaseCompleted:
		r.logger.Debugf("Volumes restore request %s has been completed", restoreRequestName)
	case volumes.RequestPhaseFailed:
		r.logger.Debugf("Volumes restore request %s has failed", restoreRequestName)
	default:
		return fmt.Errorf("invalid snapshot request phase: %s", restoreRequest.Status.Phase)
	}

	return nil
}

func (r *Restorer) reconcileInProgress(ctx context.Context, restoreRequestName string, restoreRequest *volumes.SnapshotRequest) (retErr error) {
	r.logger.Infof("Reconciling in-progress volumes restore request %s", restoreRequestName)
	if restoreRequest.Status.Phase != volumes.RequestPhaseInProgress {
		return fmt.Errorf("invalid phase for snapshot request %s, expected %s, got %s", restoreRequestName, volumes.RequestPhaseInProgress, restoreRequest.Status.Phase)
	}
	defer r.logger.Infof("Reconciled in-progress volumes restore request %s", restoreRequestName)

	continueReconciling := false
	defer func() {
		if retErr == nil {
			return
		}
		restoreRequest.Status.Phase = volumes.RequestPhaseFailed
	}()

	for _, snapshotConfig := range restoreRequest.Spec.VolumeSnapshotConfigs {
		pvcName := types.NamespacedName{
			Namespace: snapshotConfig.PersistentVolumeClaim.Namespace,
			Name:      snapshotConfig.PersistentVolumeClaim.Name,
		}.String()
		snapshotStatus, ok := restoreRequest.Status.Snapshots[pvcName]
		if !ok {
			continue
		}

		switch snapshotStatus.Phase {
		case volumes.RequestPhaseNotStarted:
			snapshotStatus.Phase = volumes.RequestPhaseInProgress
			fallthrough
		case volumes.RequestPhaseInProgress:
			newStatus, err := r.reconcileInProgressPVC(ctx, restoreRequestName, snapshotConfig, snapshotStatus)
			restoreRequest.Status.Snapshots[pvcName] = newStatus
			if err != nil {
				r.logger.Errorf("failed to reconcile in-progress volumes restore request %s for PVC %s: %v", restoreRequestName, pvcName, err)
			}
			if newStatus.Phase == volumes.RequestPhaseInProgress {
				// at least one volume snapshot creation is still in progress
				continueReconciling = true
				continue
			}
		case volumes.RequestPhaseCompleted:
			r.logger.Debugf("PVC %s has been already restored", pvcName)
		case volumes.RequestPhaseSkipped:
			r.logger.Debugf("PVC %s already exists, restore skipped", pvcName)
		case volumes.RequestPhaseFailed:
			r.logger.Errorf("Failed to restore PVC %s", pvcName)
		default:
			return fmt.Errorf("invalid restore request phase %s for PVC %s in restore snapshot request %s", snapshotStatus.Phase, pvcName, restoreRequestName)
		}
	}

	if !continueReconciling {
		restoreRequest.Status.Phase = volumes.RequestPhaseCompleted
	}
	return nil
}

func (r *Restorer) reconcileInProgressPVC(ctx context.Context, restoreRequestName string, config volumes.SnapshotConfig, restoreStatus volumes.SnapshotStatus) (status volumes.SnapshotStatus, retErr error) {
	if restoreStatus.Phase != volumes.RequestPhaseInProgress {
		return restoreStatus, fmt.Errorf("invalid phase for snapshot request %s, expected %s, got %s", restoreRequestName, volumes.RequestPhaseInProgress, restoreStatus.Phase)
	}
	status = restoreStatus
	defer func() {
		if retErr != nil {
			status.Phase = volumes.RequestPhaseFailed
		}
	}()

	// First, check if the PVC already exists
	originalPVC := &config.PersistentVolumeClaim
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

	volumeSnapshotName := fmt.Sprintf("%s-%s", config.PersistentVolumeClaim.Name, restoreRequestName)
	pvcName := types.NamespacedName{
		Namespace: config.PersistentVolumeClaim.Namespace,
		Name:      config.PersistentVolumeClaim.Name,
	}

	// Check if the pre-provisioned VolumeSnapshotContent resource exists. If it doesn't, create it.
	justCreated := false
	volumeSnapshotContent, err := r.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Get(ctx, volumeSnapshotName, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		// create new VolumeSnapshotContent
		volumeSnapshotContent, err = r.createVolumeSnapshotContentResource(ctx, restoreRequestName, volumeSnapshotName, config, restoreStatus)
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
		volumeSnapshot, err = r.createVolumeSnapshotResource(ctx, restoreRequestName, volumeSnapshotName, pvcName, config.VolumeSnapshotClassName)
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
	originalPersistentVolumeClaim := config.PersistentVolumeClaim
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

	status.Phase = volumes.RequestPhaseCompleted
	r.logger.Infof(
		"Restored PersistentVolumeClaim %s/%s from VolumeSnapshot %s/%s",
		restoredPersistentVolumeClaim.Namespace, restoredPersistentVolumeClaim.Name,
		volumeSnapshot.Namespace, volumeSnapshot.Name)

	return status, nil
}

// createVolumeSnapshotResource creates the pre-provisioned VolumeSnapshot from which the PVC will be restored
func (r *Restorer) createVolumeSnapshotResource(ctx context.Context, restoreRequestName, volumeSnapshotName string, pvcName types.NamespacedName, volumeSnapshotClassName string) (*snapshotsv1api.VolumeSnapshot, error) {
	r.logger.Debugf("Create VolumeSnapshot %s for PersistentVolumeClaim %s for restore request %s", volumeSnapshotName, pvcName.String(), restoreRequestName)

	volumeSnapshot := &snapshotsv1api.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pvcName.Namespace,
			Name:      volumeSnapshotName,
			Labels: map[string]string{
				meta.RestoreRequestLabel:       restoreRequestName,
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
	volumeSnapshot, err = r.snapshotsClient.SnapshotV1().VolumeSnapshots(pvcName.Namespace).Create(ctx, volumeSnapshot, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not create VolumeSnapshot resource for the PersistentVolumeClaim %s: %w", pvcName, err)
	}
	r.logger.Infof("Created VolumeSnapshot resource %s/%s for the PersistentVolumeClaim %s", volumeSnapshot.Namespace, volumeSnapshot.Name, pvcName)

	return volumeSnapshot, nil
}

// createVolumeSnapshotResource creates the pre-provisioned VolumeSnapshotContent from which the PVC will be restored
func (r *Restorer) createVolumeSnapshotContentResource(ctx context.Context, restoreRequestName, volumeSnapshotName string, config volumes.SnapshotConfig, restoreStatus volumes.SnapshotStatus) (*snapshotsv1api.VolumeSnapshotContent, error) {
	r.logger.Debugf(
		"Create VolumeSnapshotContent %s for PersistentVolumeClaim %s/%s for restore request %s",
		volumeSnapshotName,
		config.PersistentVolumeClaim.Namespace,
		config.PersistentVolumeClaim.Name,
		restoreRequestName)

	volumeSnapshotContent := &snapshotsv1api.VolumeSnapshotContent{
		ObjectMeta: metav1.ObjectMeta{
			Name: volumeSnapshotName,
			Labels: map[string]string{
				meta.RestoreRequestLabel:       restoreRequestName,
				persistentVolumeClaimNameLabel: config.PersistentVolumeClaim.Name,
			},
		},
		Spec: snapshotsv1api.VolumeSnapshotContentSpec{
			DeletionPolicy: snapshotsv1api.VolumeSnapshotContentRetain,
			Driver:         config.CSIDriver,
			Source: snapshotsv1api.VolumeSnapshotContentSource{
				SnapshotHandle: ptr.To(restoreStatus.SnapshotHandle),
			},
			VolumeSnapshotRef: corev1.ObjectReference{
				Name:      volumeSnapshotName,
				Namespace: config.PersistentVolumeClaim.Namespace,
			},
		},
	}
	if config.VolumeSnapshotClassName != "" {
		volumeSnapshotContent.Spec.VolumeSnapshotClassName = &config.VolumeSnapshotClassName
	}
	if config.PersistentVolumeClaim.Spec.VolumeMode != nil {
		volumeSnapshotContent.Spec.SourceVolumeMode = config.PersistentVolumeClaim.Spec.VolumeMode
	}

	var err error
	volumeSnapshotContent, err = r.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Create(ctx, volumeSnapshotContent, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf(
			"could not create VolumeSnapshotContent resource for the PersistentVolumeClaim %s/%s: %w",
			config.PersistentVolumeClaim.Namespace,
			config.PersistentVolumeClaim.Name,
			err)
	}
	r.logger.Infof("Created VolumeSnapshotContent resource %s for the PersistentVolumeClaim %s/%s",
		volumeSnapshotContent.Name,
		config.PersistentVolumeClaim.Namespace,
		config.PersistentVolumeClaim.Name)

	return volumeSnapshotContent, nil
}
