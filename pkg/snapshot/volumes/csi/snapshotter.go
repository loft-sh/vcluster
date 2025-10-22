package csi

import (
	"context"
	"errors"
	"fmt"

	snapshotsv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

const (
	VolumeSnapshotsAnnotation         = "vcluster.loft.sh/volumesnapshots"
	dynamicVolumeSnapshotLabel        = "vcluster.loft.sh/dynamicvolumesnapshot"
	PreProvisionedVolumeSnapshotLabel = "vcluster.loft.sh/preprovisionedvolumesnapshot"
	persistentVolumeClaimNameLabel    = "vcluster.loft.sh/persistentvolumeclaim"
)

var (
	ErrVolumeSnapshotConfigNotValid = errors.New("volume snapshots config is not valid")
	ErrVolumeSnapshotClassNotFound  = errors.New("VolumeSnapshotClass error")
)

// VolumeSnapshotter is a volume.Snapshotter interface implementation that creates CSI volume snapshots.
type VolumeSnapshotter struct {
	snapshotHandler
	vConfig *config.VirtualClusterConfig
}

// NewVolumeSnapshotter creates a new instance of the CSI volume snapshotter.
func NewVolumeSnapshotter(vConfig *config.VirtualClusterConfig, kubeClient *kubernetes.Clientset, snapshotsClient *snapshotsv1.Clientset, eventRecorder record.EventRecorder, logger loghelper.Logger) (*VolumeSnapshotter, error) {
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

	snapshotter := &VolumeSnapshotter{
		snapshotHandler: snapshotHandler{
			kubeClient:      kubeClient,
			snapshotsClient: snapshotsClient,
			eventRecorder:   eventRecorder,
			logger:          logger,
		},
		vConfig: vConfig,
	}
	return snapshotter, nil
}

// CheckIfPersistentVolumeIsSupported checks if the snapshotter can create a CSI volume snapshot of
// the specified persistent volume.
func (s *VolumeSnapshotter) CheckIfPersistentVolumeIsSupported(pv *corev1.PersistentVolume) error {
	hasPersistentVolumeClaim := pv.Spec.ClaimRef != nil &&
		pv.Spec.ClaimRef.Name != "" &&
		pv.Spec.ClaimRef.Namespace != "" &&
		pv.Spec.ClaimRef.Kind == "PersistentVolumeClaim"
	if !hasPersistentVolumeClaim {
		return fmt.Errorf("specified PersistentVolume does not have a PersistentVolumeClaim set: %w", volumes.ErrPersistentVolumeNotSupported)
	}

	managedByCSIDriver := pv.Spec.CSI != nil && pv.Spec.CSI.Driver != ""
	if !managedByCSIDriver {
		return fmt.Errorf("specified PersistentVolume is not managed by the CSI driver: %w", volumes.ErrPersistentVolumeNotSupported)
	}

	return nil
}

func (s *VolumeSnapshotter) Reconcile(ctx context.Context, requestObj runtime.Object, requestName string, request *volumes.SnapshotsRequest, status *volumes.SnapshotsStatus) error {
	s.logger.Debugf("Reconcile volume snapshots for snapshot request %s", requestName)
	defer s.logger.Debugf("Reconciled volume snapshots for snapshot request %s", requestName)
	var err error

	switch status.Phase {
	case volumes.RequestPhaseNotStarted:
		err = s.reconcileNotStarted(ctx, requestName, request, status)
		if err != nil {
			return fmt.Errorf("failed to reconcile new volumes snapshot request %s: %w", requestName, err)
		}
	case volumes.RequestPhaseInProgress:
		err = s.reconcileInProgress(ctx, requestObj, requestName, request, status)
		if err != nil {
			return fmt.Errorf("failed to reconcile volumes snapshots request %s: %w", requestName, err)
		}
	case volumes.RequestPhaseCompleted:
		fallthrough
	case volumes.RequestPhasePartiallyFailed:
		fallthrough
	case volumes.RequestPhaseFailed:
		fallthrough
	case volumes.RequestPhaseCanceled:
		fallthrough
	case volumes.RequestPhaseDeleted:
		fallthrough
	case volumes.RequestPhaseSkipped:
		err = s.reconcileDone(ctx, requestName, status)
		if err != nil {
			return fmt.Errorf("failed to reconcile failed volumes snapshot request %s: %w", requestName, err)
		}
	case volumes.RequestPhaseDeleting:
		fallthrough
	case volumes.RequestPhaseCanceling:
		err = s.reconcileDeleting(ctx, requestObj, requestName, request, status)
		if err != nil {
			return fmt.Errorf("failed to reconcile canceling volumes snapshot request %s: %w", requestName, err)
		}
	default:
		return fmt.Errorf("invalid snapshot request phase: %s", status.Phase)
	}

	return nil
}

func (s *VolumeSnapshotter) Cleanup(ctx context.Context) error {
	s.logger.Debugf("Delete pre-provisioned VolumeSnapshots and VolumeSnapshotContents resources")

	// get all volume snapshots by label
	listOptions := metav1.ListOptions{
		LabelSelector: PreProvisionedVolumeSnapshotLabel,
	}

	// 1. Delete all VolumeSnapshot resources that have been created while creating vcluster snapshot.
	//
	// Currently, all VolumeSnapshot resources are using a VolumeSnapshotClass with delete policy set to
	// 'Retain', so it's safe to delete the VolumeSnapshots that have been already  added to a vcluster
	// snapshot.
	volumeSnapshots, err := s.snapshotsClient.SnapshotV1().VolumeSnapshots("").List(ctx, listOptions)
	if err != nil {
		return fmt.Errorf("failed to list VolumeSnapshots: %w", err)
	}
	for _, volumeSnapshot := range volumeSnapshots.Items {
		s.logger.Debugf("Delete pre-provisioned VolumeSnapshot %s/%s", volumeSnapshot.Namespace, volumeSnapshot.Name)
		err = s.snapshotsClient.SnapshotV1().VolumeSnapshots(volumeSnapshot.Namespace).Delete(ctx, volumeSnapshot.Name, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete pre-provisioned VolumeSnapshot %s/%s: %w", volumeSnapshot.Name, volumeSnapshot.Name, err)
		}
		s.logger.Debugf("Deleted pre-provisioned VolumeSnapshot %s/%s", volumeSnapshot.Namespace, volumeSnapshot.Name)
	}

	// 2. Delete all VolumeSnapshotContent resources that have been created while creating vcluster snapshot.
	volumeSnapshotContents, err := s.snapshotsClient.SnapshotV1().VolumeSnapshotContents().List(ctx, listOptions)
	if err != nil {
		return fmt.Errorf("failed to list VolumeSnapshotContents: %w", err)
	}
	for _, volumeSnapshotContent := range volumeSnapshotContents.Items {
		s.logger.Debugf("Delete pre-provisioned VolumeSnapshotContent %s", volumeSnapshotContent.Name)
		err = s.snapshotsClient.SnapshotV1().VolumeSnapshotContents().Delete(ctx, volumeSnapshotContent.Name, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete VolumeSnapshotContent %s: %w", volumeSnapshotContent.Name, err)
		}
		s.logger.Debugf("Deleted pre-provisioned VolumeSnapshotContent %s", volumeSnapshotContent.Name)
	}

	s.logger.Debugf("Deleted pre-provisioned VolumeSnapshots and VolumeSnapshotContents resources")
	return nil
}
