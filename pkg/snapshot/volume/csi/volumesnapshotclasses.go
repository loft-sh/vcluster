package csi

import (
	"context"
	"fmt"
	snapshotsv1api "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
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
