package csi

import (
	"context"
	"fmt"
	"slices"

	"k8s.io/apimachinery/pkg/types"

	snapshotsv1api "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *VolumeSnapshotter) mapCSIDriversToVolumeSnapshotClasses(ctx context.Context) (map[string][]string, error) {
	m := map[string][]string{}

	volumeSnapshotClasses, err := s.snapshotsClient.SnapshotV1().VolumeSnapshotClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list VolumeSnapshotClasses: %w", err)
	}
	for _, volumeSnapshotClass := range volumeSnapshotClasses.Items {
		if volumeSnapshotClass.DeletionPolicy == snapshotsv1api.VolumeSnapshotContentRetain {
			m[volumeSnapshotClass.Driver] = append(m[volumeSnapshotClass.Driver], volumeSnapshotClass.Name)
		}
		s.logger.Debugf("Found VolumeSnapshotClass %q (with '%s' deletion policy) for CSI driver %q", volumeSnapshotClass.Name, volumeSnapshotClass.DeletionPolicy, volumeSnapshotClass.Driver)
	}

	return m, nil
}

func (s *VolumeSnapshotter) getVolumeSnapshotClass(pv *corev1.PersistentVolume, allClassesForDriver []string) (string, error) {
	// 1. Check if a VolumeSnapshotClass has been configured for the PersistentVolumeClaim
	pvcName := types.NamespacedName{
		Namespace: pv.Spec.ClaimRef.Namespace,
		Name:      pv.Spec.ClaimRef.Name,
	}
	pvcConfig, ok := s.vConfig.Experimental.CSIVolumeSnapshots.ByPersistentVolumeClaim[pvcName.String()]
	if ok {
		if pvcConfig.VolumeSnapshotClass == "" {
			return "", fmt.Errorf("VolumeSnapshotClass is not correctly set for PersistentVolumeClaim %s volume snapshots config, check your vCluster config: %w", pvcName, ErrVolumeSnapshotConfigNotValid)
		}
		if slices.Contains(allClassesForDriver, pvcConfig.VolumeSnapshotClass) {
			return pvcConfig.VolumeSnapshotClass, nil
		}
		return "", fmt.Errorf("VolumeSnapshotClass %s with delete policy 'Retain', which is configured for PVC %s, is not found: %w", pvcConfig.VolumeSnapshotClass, pvcName, ErrVolumeSnapshotClassNotFound)
	}

	// 2. Check if a VolumeSnapshotClass has been configured for the CSI driver
	driverConfig, ok := s.vConfig.Experimental.CSIVolumeSnapshots.ByDriver[pv.Spec.CSI.Driver]
	if !ok {
		return "", fmt.Errorf("volume snapshots are not configured for CSI driver %s", pv.Spec.CSI.Driver)
	}

	if slices.Contains(allClassesForDriver, driverConfig.VolumeSnapshotClass) {
		return driverConfig.VolumeSnapshotClass, nil
	}

	// VolumeSnapshotClass has not been configured
	return "", fmt.Errorf("VolumeSnapshotClass %s with delete policy 'Retain', which is configured for CSI driver %s, is not found: %w", driverConfig.VolumeSnapshotClass, pv.Spec.CSI.Driver, ErrVolumeSnapshotClassNotFound)
}
