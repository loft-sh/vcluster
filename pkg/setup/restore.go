package setup

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned"
	"github.com/loft-sh/log"
	"k8s.io/client-go/kubernetes"

	"github.com/loft-sh/vcluster/pkg/snapshot/volume/csi"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
)

func RestoreVolumes(controllerCtx *synccontext.ControllerContext, logger log.Logger) error {
	logger.Info("Restore volumes from VolumeSnapshots")
	restConfig := controllerCtx.VirtualManager.GetConfig()
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("could not create kube client: %w", err)
	}
	snapshotClient, err := snapshotv1.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("could not create snapshot client: %w", err)
	}
	csiVolumeRestorer, err := csi.NewVolumeRestorer(controllerCtx.Config, kubeClient, snapshotClient, logger)
	if err != nil {
		return fmt.Errorf("could not create csi VolumeRestorer: %w", err)
	}

	listOptions := metav1.ListOptions{
		LabelSelector: csi.PreProvisionedVolumeSnapshotLabel,
	}
	volumeSnapshots, err := snapshotClient.SnapshotV1().VolumeSnapshots("").List(controllerCtx.Context, listOptions)
	if err != nil {
		return fmt.Errorf("could not list VolumeSnapshots: %w", err)
	}

	err = csiVolumeRestorer.RestoreVolumes(controllerCtx.Context, volumeSnapshots.Items)
	if err != nil {
		return fmt.Errorf("could not restore volumes from VolumeSnapshots: %w", err)
	}

	logger.Info("Restored volumes from VolumeSnapshots")
	return nil
}
