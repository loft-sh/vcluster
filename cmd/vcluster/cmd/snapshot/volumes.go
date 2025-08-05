package snapshot

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned"
	"github.com/loft-sh/log"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/snapshot/volume"
	"github.com/loft-sh/vcluster/pkg/snapshot/volume/auto"
	"github.com/loft-sh/vcluster/pkg/snapshot/volume/csi"
	"github.com/loft-sh/vcluster/pkg/snapshot/volume/filesystem"
)

func CreateVolumeSnapshots(ctx context.Context, volumeSnapshotter volume.Snapshotter, kubeClient *kubernetes.Clientset) (volume.CreateSnapshotsResult, error) {
	// get all PVs
	pvs, err := kubeClient.CoreV1().PersistentVolumes().List(ctx, v1.ListOptions{})
	if err != nil {
		return volume.CreateSnapshotsResult{}, fmt.Errorf("could not list PersistentVolumes: %w", err)
	}

	// Try creating snapshots for all PVs
	result, err := volumeSnapshotter.CreateSnapshots(ctx, pvs.Items)
	if err != nil {
		return volume.CreateSnapshotsResult{}, fmt.Errorf("could not create volume snapshots: %w", err)
	}

	return result, nil
}

func IsPvOrPvcWithSnapshot(etcdObjectKey string, snapshottedPvcs, snapshottedPvs map[string]struct{}) bool {
	const (
		// TODO check if vcluster always uses prefix '/registry' for etcd keys
		pvPrefix  = "/registry/persistentvolumes/"
		pvcPrefix = "/registry/persistentvolumeclaims/"
	)
	if strings.HasPrefix(etcdObjectKey, pvPrefix) {
		pvName := strings.TrimPrefix(etcdObjectKey, pvPrefix)
		if _, ok := snapshottedPvs[pvName]; ok {
			return true
		}
	} else if strings.HasPrefix(etcdObjectKey, pvcPrefix) {
		pvcName := strings.TrimPrefix(etcdObjectKey, pvcPrefix)
		if _, ok := snapshottedPvcs[pvcName]; ok {
			return true
		}
	}
	return false
}

func CreateVirtualKubeClients(config *config.VirtualClusterConfig) (*kubernetes.Clientset, *versioned.Clientset, error) {
	// read kubeconfig
	out, err := os.ReadFile(config.VirtualClusterKubeConfig().KubeConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("could not read kubeconfig file: %w", err)
	}
	clientConfig, err := clientcmd.NewClientConfigFromBytes(out)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create a client config from kubeconfig: %w", err)
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("could not create a rest client config: %w", err)
	}

	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create kube client: %w", err)
	}

	snapshotClient, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create snapshot client: %w", err)
	}

	return kubeClient, snapshotClient, nil
}

func CreateVolumeSnapshotter(ctx context.Context, vConfig *config.VirtualClusterConfig, kubeClient *kubernetes.Clientset, snapshotsClient *versioned.Clientset, etcdSnapshotLocation string, logger log.Logger) (volume.Snapshotter, error) {
	csiVolumeSnapshotter, err := csi.NewVolumeSnapshotter(ctx, vConfig, kubeClient, snapshotsClient, etcdSnapshotLocation, logger)
	if err != nil {
		logger.Errorf("could not create CSI volume snapshotter, CSI VolumeSnapshots will not be created: %v", err)
	}
	filesystemSnapshotter, err := filesystem.NewVolumeSnapshotter(vConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("could not create filesystem snapshotter: %w", err)
	}

	var snapshotters []volume.Snapshotter
	if csiVolumeSnapshotter != nil {
		// Use CSI volume snapshot only when it was successfully created.
		// e.g. the CSI VolumeSnapshotter creation will fail when volume snapshot CRDs are not
		// installed in the virtual cluster, so in that case the snapshot command will just not
		// try to use CSI volume snapshotter for PVCs, and it will only use the file-system
		// snapshotter.
		snapshotters = append(snapshotters, csiVolumeSnapshotter)
	}
	snapshotters = append(snapshotters, filesystemSnapshotter)
	autoSnapshotter, err := auto.NewVolumeSnapshotter(logger, snapshotters...)
	if err != nil {
		return nil, fmt.Errorf("could not create auto snapshotter: %w", err)
	}
	return autoSnapshotter, nil
}
