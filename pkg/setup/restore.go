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
	logger.Info("Start restoring volumes from VolumeSnapshots")
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

	// TODO: discuss in which of the following ways to restore the volumes:
	//  1. Current approach here: asynchronously in a goroutine, so it does not block the cluster from starting;
	//     Pro: Virtual cluster start is not blocked.
	//     Con: Cancelling the restore process on-demand.
	//  2. Synchronously in a goroutine, so it waits for all volumes restores to finish before starting the cluster;
	//     Pro: Easier to cancel the restore process on-demand.
	//     Con: Waiting too much for the virtual cluster to start.
	//  3. In a new TBA VolumeRestoreController?
	//     Pro: Restore process can be orchestrated in more advanced ways.
	//     Con: Requires implementing a new controller. What to reconcile in the controller?
	go func() {
		err = csiVolumeRestorer.RestoreVolumes(controllerCtx.Context, volumeSnapshots.Items)
		if err != nil {
			logger.Errorf("failed to restore volumes from VolumeSnapshots: %v", err)
		}
	}()

	logger.Info("Started restoring volumes from VolumeSnapshots")
	return nil
}
