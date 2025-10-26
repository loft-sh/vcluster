package csi

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *VolumeSnapshotter) reconcileNotStarted(ctx context.Context, requestName string, request *volumes.SnapshotsRequest, status *volumes.SnapshotsStatus) error {
	s.logger.Infof("Reconciling new volume snapshots request %s", requestName)
	if status.Phase != volumes.RequestPhaseNotStarted {
		return fmt.Errorf("invalid phase for snapshot request %s, expected %s, got %s", requestName, volumes.RequestPhaseNotStarted, status.Phase)
	}
	defer s.logger.Infof("Reconciled new volume snapshots request %s", requestName)

	// first get all PersistentVolumeClaims
	// for private nodes, we need to get all PVCs in the virtual cluster
	// for share nodes, we need to get only PVCs in the vCluster host namespace
	var pvcsNamespace string
	var listOptions v1.ListOptions
	if !s.vConfig.PrivateNodes.Enabled {
		// if private nodes are disabled, we need to get all PVCs in all namespaces
		pvcsNamespace = s.vConfig.HostNamespace
		listOptions = v1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", translate.MarkerLabel, s.vConfig.Name),
		}
	}
	pvcs, err := s.kubeClient.CoreV1().PersistentVolumeClaims(pvcsNamespace).List(ctx, listOptions)
	if err != nil {
		return fmt.Errorf("failed to list PersistentVolumeClaims: %w", err)
	}

	var volumeSnapshotRequests []volumes.SnapshotRequest
	for _, pvc := range pvcs.Items {
		if pvc.Spec.VolumeName == "" {
			// PVC is not bound to a PV, skip it
			continue
		}
		pv, err := s.kubeClient.CoreV1().PersistentVolumes().Get(ctx, pvc.Spec.VolumeName, v1.GetOptions{})
		if err != nil {
			s.logger.Errorf("failed to get PersistentVolume %s for PersistentVolumeClaim %s/%s: %w", pvc.Spec.VolumeName, pvc.Namespace, pvc.Name, err)
			continue
		}

		// check if creating a snapshot for this PV is supported
		err = s.CheckIfPersistentVolumeIsSupported(pv)
		if err != nil {
			s.logger.Infof("Skip creating a snapshot for PersistentVolume %s, since it is not supported: %v", pv.Name, err)
			continue
		}
		volumeSnapshotRequest := volumes.SnapshotRequest{
			CSIDriver: pv.Spec.CSI.Driver,
		}
		pvcCopy := pvc.DeepCopy()
		delete(pvcCopy.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
		delete(pvcCopy.Annotations, "pv.kubernetes.io/bind-completed")
		delete(pvcCopy.Annotations, "pv.kubernetes.io/bound-by-controller")
		delete(pvcCopy.Annotations, "volume.beta.kubernetes.io/storage-provisioner")
		delete(pvcCopy.Annotations, "volume.kubernetes.io/storage-provisioner")
		pvcCopy.ManagedFields = nil
		pvcCopy.Status = corev1.PersistentVolumeClaimStatus{}
		volumeSnapshotRequest.PersistentVolumeClaim = *pvcCopy

		if volumeSnapshotClassName, ok := pvc.Labels[volumes.SnapshotClassNameLabel]; ok {
			volumeSnapshotRequest.VolumeSnapshotClassName = volumeSnapshotClassName
		}

		volumeSnapshotRequests = append(volumeSnapshotRequests, volumeSnapshotRequest)
	}

	// Snapshot request successfully initialized, update phase to InProgress
	request.Requests = volumeSnapshotRequests
	status.Snapshots = map[string]volumes.SnapshotStatus{}
	status.Phase = volumes.RequestPhaseInProgress
	return nil
}
