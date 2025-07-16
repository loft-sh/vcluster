package auto

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/loft-sh/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	
	"github.com/loft-sh/vcluster/pkg/snapshot/volume"
)

type VolumeSnapshotter struct {
	logger       log.Logger
	snapshotters []volume.Snapshotter
}

func NewVolumeSnapshotter(logger log.Logger, snapshotters ...volume.Snapshotter) (*VolumeSnapshotter, error) {
	if len(snapshotters) == 0 {
		return nil, errors.New("at least one volume snapshotter must be set")
	}
	if logger == nil {
		return nil, errors.New("logger must be set")
	}
	snapshotter := &VolumeSnapshotter{
		logger:       logger,
		snapshotters: snapshotters,
	}
	return snapshotter, nil
}

func (s *VolumeSnapshotter) CheckIfPersistentVolumeIsSupported(pv *corev1.PersistentVolume) error {
	var allErrors []error
	for _, snapshotter := range s.snapshotters {
		err := snapshotter.CheckIfPersistentVolumeIsSupported(pv)
		if err != nil {
			allErrors = append(allErrors, err)
		}
	}

	if len(allErrors) > 0 {
		pvNamespacedName := types.NamespacedName{
			Name:      pv.Name,
			Namespace: pv.Namespace,
		}
		return fmt.Errorf("none of the snapshotters supports PersistentVolume %s: %w", pvNamespacedName, errors.Join(allErrors...))
	}

	return nil
}

func (s *VolumeSnapshotter) CreateSnapshots(ctx context.Context, persistentVolumes []corev1.PersistentVolume) error {
	s.logger.Info("Start creating volume snapshots")
	defer s.logger.Info("Finished creating volume snapshots")

	persistentVolumesPerSnapshotter := make([][]corev1.PersistentVolume, len(s.snapshotters))
	unsupportedPersistentVolumes := map[string]error{}

	for _, pv := range persistentVolumes {
		persistentVolumeSupported := false
		var persistentVolumeSupportCheckErrors []error

		for i, snapshotter := range s.snapshotters {
			err := snapshotter.CheckIfPersistentVolumeIsSupported(&pv)
			if err != nil {
				persistentVolumeSupportCheckErrors = append(persistentVolumeSupportCheckErrors, err)
			} else {
				// snapshotter supports persistent volume
				persistentVolumesPerSnapshotter[i] = append(persistentVolumesPerSnapshotter[i], pv)
				persistentVolumeSupported = true
				break
			}
		}
		if !persistentVolumeSupported {
			unsupportedPersistentVolumes[pv.Name] = errors.Join(persistentVolumeSupportCheckErrors...)
		}
	}

	for unsupportedPersistentVolume, checkError := range unsupportedPersistentVolumes {
		s.logger.Warnf(
			"PersistentVolume %s is not supported by any of the snapshotters in the auto volume snapshotter: %v",
			unsupportedPersistentVolume,
			checkError)
	}

	var wg sync.WaitGroup
	maxSnapshotters := len(s.snapshotters)
	errCh := make(chan error, maxSnapshotters)

	for i, snapshotter := range s.snapshotters {
		supportedPersistentVolumes := persistentVolumesPerSnapshotter[i]
		if len(supportedPersistentVolumes) == 0 {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := snapshotter.CreateSnapshots(ctx, supportedPersistentVolumes)
			if err != nil {
				errCh <- err
			}
		}()
	}

	// wait for all snapshots to be taken and close the errors channel
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
