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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
)

type Restorer struct {
	snapshotHandler
	vConfig *config.VirtualClusterConfig
}

func NewRestorer(vConfig *config.VirtualClusterConfig, kubeClient *kubernetes.Clientset, snapshotsClient *snapshotsv1.Clientset, eventRecorder record.EventRecorder, logger loghelper.Logger) (*Restorer, error) {
	if vConfig == nil {
		return nil, errors.New("virtual cluster config is required")
	}
	if kubeClient == nil {
		return nil, errors.New("kubernetes client is required")
	}
	if snapshotsClient == nil {
		return nil, errors.New("snapshot client is required")
	}
	if eventRecorder == nil {
		return nil, errors.New("event recorder is required")
	}
	if logger == nil {
		return nil, errors.New("logger is required")
	}

	restorer := &Restorer{
		snapshotHandler: snapshotHandler{
			kubeClient:      kubeClient,
			snapshotsClient: snapshotsClient,
			eventRecorder:   eventRecorder,
			logger:          logger,
		},
		vConfig: vConfig,
	}
	return restorer, nil
}

// Reconcile volumes restore request.
func (r *Restorer) Reconcile(ctx context.Context, requestObj runtime.Object, requestName string, request *volumes.RestoreRequestSpec, status *volumes.RestoreRequestStatus) error {
	r.logger.Infof("Restore volumes for restore request %s", requestName)
	var err error

	switch status.Phase {
	case volumes.RequestPhaseNotStarted:
		status.Phase = volumes.RequestPhaseInProgress
		fallthrough
	case volumes.RequestPhaseInProgress:
		err = r.reconcileInProgress(ctx, requestObj, requestName, request, status)
		if err != nil {
			return fmt.Errorf("failed to reconcile failed volumes snapshot request %s: %w", requestName, err)
		}
	case volumes.RequestPhaseCompleted:
		r.logger.Debugf("Volumes restore request %s has been completed", requestName)
	case volumes.RequestPhaseFailed:
		r.logger.Debugf("Volumes restore request %s has failed", requestName)
	default:
		return fmt.Errorf("invalid snapshot request phase: %s", status.Phase)
	}

	return nil
}

func (r *Restorer) reconcileInProgress(ctx context.Context, requestObj runtime.Object, requestName string, request *volumes.RestoreRequestSpec, status *volumes.RestoreRequestStatus) (retErr error) {
	r.logger.Infof("Reconciling in-progress volumes restore request %s", requestName)
	if status.Phase != volumes.RequestPhaseInProgress {
		return fmt.Errorf("invalid phase for snapshot request %s, expected %s, got %s", requestName, volumes.RequestPhaseInProgress, status.Phase)
	}
	defer r.logger.Infof("Reconciled in-progress volumes restore request %s", requestName)

	continueReconciling := false
	defer func() {
		if retErr == nil {
			return
		}
		status.Phase = volumes.RequestPhaseFailed
		status.Error.Message = retErr.Error()
	}()

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
			if newStatus.Phase == volumes.RequestPhaseInProgress {
				// at least one volume restore is still in progress
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
			return fmt.Errorf("invalid restore request phase %s for PVC %s in restore snapshot request %s", volumeRestoreStatus.Phase, pvcName, requestName)
		}
	}

	if !continueReconciling {
		status.Phase = volumes.RequestPhaseCompleted
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
			status.Phase = volumes.RequestPhaseFailed
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
	volumeSnapshotContent, err := r.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Get(ctx, volumeSnapshotName, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		// create new VolumeSnapshotContent
		volumeSnapshotContent, err = r.createVolumeSnapshotContentResource(ctx, requestName, volumeSnapshotName, volumeRestoreRequest)
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
		volumeSnapshot, err = r.createVolumeSnapshotResource(ctx, requestName, volumeSnapshotName, pvcName, volumeRestoreRequest.VolumeSnapshotClassName)
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

	status.Phase = volumes.RequestPhaseCompleted
	r.logger.Infof(
		"Restored PersistentVolumeClaim %s/%s from VolumeSnapshot %s/%s",
		restoredPersistentVolumeClaim.Namespace, restoredPersistentVolumeClaim.Name,
		volumeSnapshot.Namespace, volumeSnapshot.Name)

	return status, nil
}

// createVolumeSnapshotResource creates the pre-provisioned VolumeSnapshot from which the PVC will be restored
func (r *Restorer) createVolumeSnapshotResource(ctx context.Context, requestName, volumeSnapshotName string, pvcName types.NamespacedName, volumeSnapshotClassName string) (*snapshotsv1api.VolumeSnapshot, error) {
	r.logger.Debugf("Create VolumeSnapshot %s for PersistentVolumeClaim %s for restore request %s", volumeSnapshotName, pvcName.String(), requestName)

	volumeSnapshot := &snapshotsv1api.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pvcName.Namespace,
			Name:      volumeSnapshotName,
			Labels: map[string]string{
				meta.RestoreRequestLabel:       requestName,
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
func (r *Restorer) createVolumeSnapshotContentResource(ctx context.Context, requestName, volumeSnapshotName string, volumeRestoreRequest volumes.RestoreRequest) (*snapshotsv1api.VolumeSnapshotContent, error) {
	r.logger.Debugf(
		"Create VolumeSnapshotContent %s for PersistentVolumeClaim %s/%s for restore request %s",
		volumeSnapshotName,
		volumeRestoreRequest.PersistentVolumeClaim.Namespace,
		volumeRestoreRequest.PersistentVolumeClaim.Name,
		requestName)

	volumeSnapshotContent := &snapshotsv1api.VolumeSnapshotContent{
		ObjectMeta: metav1.ObjectMeta{
			Name: volumeSnapshotName,
			Labels: map[string]string{
				meta.RestoreRequestLabel:       requestName,
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
	volumeSnapshotContent, err = r.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Create(ctx, volumeSnapshotContent, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf(
			"could not create VolumeSnapshotContent resource for the PersistentVolumeClaim %s/%s: %w",
			volumeRestoreRequest.PersistentVolumeClaim.Namespace,
			volumeRestoreRequest.PersistentVolumeClaim.Name,
			err)
	}
	r.logger.Infof("Created VolumeSnapshotContent resource %s for the PersistentVolumeClaim %s/%s",
		volumeSnapshotContent.Name,
		volumeRestoreRequest.PersistentVolumeClaim.Namespace,
		volumeRestoreRequest.PersistentVolumeClaim.Name)

	return volumeSnapshotContent, nil
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
