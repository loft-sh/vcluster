package filesystem

import (
	"context"
	"errors"

	"github.com/loft-sh/log"
	corev1 "k8s.io/api/core/v1"

	"github.com/loft-sh/vcluster/pkg/config"
)

type VolumeSnapshotter struct {
	vConfig *config.VirtualClusterConfig
	logger  log.Logger
}

func NewVolumeSnapshotter(vConfig *config.VirtualClusterConfig, logger log.Logger) (*VolumeSnapshotter, error) {
	if vConfig == nil {
		return nil, errors.New("virtual cluster config is nil")
	}
	if logger == nil {
		return nil, errors.New("logger is nil")
	}

	snapshotter := &VolumeSnapshotter{
		vConfig: vConfig,
		logger:  logger,
	}
	return snapshotter, nil
}

func (s *VolumeSnapshotter) CheckIfPersistentVolumeIsSupported(pv *corev1.PersistentVolume) error {
	// TODO implement me
	// File-system snapshotter should support any persistent volume. The only possible limitation
	// could be that the volume has been mounted to a pod.
	return nil
}

func (s *VolumeSnapshotter) CreateSnapshots(ctx context.Context, persistentVolumes []corev1.PersistentVolume) error {
	//TODO implement me
	for _, pv := range persistentVolumes {
		s.logger.Infof("Skipped creating file-system snapshot for PersistentVolume %s because file-system snapshots have not been implemented yet.", pv.Name)
	}
	return nil
}
