package csi

import (
	"context"
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/loft-sh/vcluster/pkg/snapshot/volumes"
)

func (s *VolumeSnapshotter) reconcileNotStarted(ctx context.Context, snapshotRequestName string, snapshotRequest *volumes.SnapshotRequest) error {
	s.logger.Infof("Reconciling new volume snapshots request %s", snapshotRequestName)
	if snapshotRequest.Status.Phase != volumes.RequestPhaseNotStarted {
		return fmt.Errorf("invalid phase for snapshot request %s, expected %s, got %s", snapshotRequestName, volumes.RequestPhaseNotStarted, snapshotRequest.Status.Phase)
	}
	defer s.logger.Infof("Reconciled new volume snapshots request %s", snapshotRequestName)

	// first get all persistent volumes
	pvs, err := s.kubeClient.CoreV1().PersistentVolumes().List(ctx, v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list persistent volumes: %w", err)
	}

	var snapshotConfigs volumes.SnapshotConfigs
	for _, pv := range pvs.Items {
		// check if creating a snapshot for this PV is supported
		err = s.CheckIfPersistentVolumeIsSupported(&pv)
		if err != nil {
			s.logger.Infof("Creating snapshot for PersistentVolume %s is not supported, skipping", pv.Name)
			continue
		}
		snapshotConfig := volumes.SnapshotConfig{
			CSIDriver: pv.Spec.CSI.Driver,
		}

		pvc, err := s.kubeClient.CoreV1().PersistentVolumeClaims(pv.Spec.ClaimRef.Namespace).Get(ctx, pv.Spec.ClaimRef.Name, v1.GetOptions{})
		if err != nil {
			s.logger.Errorf("failed to get PersistentVolumeClaim %s/%s for PersistentVolume %s: %w", pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name, pv.Name, err)
			continue
		}
		snapshotConfig.PersistentVolumeClaim = *pvc

		if volumeSnapshotClassName, ok := pvc.Labels[volumes.SnapshotClassNameLabel]; ok {
			snapshotConfig.VolumeSnapshotClassName = volumeSnapshotClassName
		}

		snapshotConfigs = append(snapshotConfigs, snapshotConfig)
	}

	// Snapshot request successfully initialized, update phase to InProgress
	snapshotRequest.Spec.VolumeSnapshotConfigs = snapshotConfigs
	snapshotRequest.Status.Phase = volumes.RequestPhaseInProgress
	return nil
}
