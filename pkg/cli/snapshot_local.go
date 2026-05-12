package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/loft-sh/log"
	vclusterconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/snapshot/pod"
	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/loft-sh/vcluster/pkg/cli/find"
)

// dataMountPath is where the vCluster data PVC is mounted in the syncer container.
const dataMountPath = "/data"

func snapshotToLocalFile(ctx context.Context, vCluster *find.VCluster,
	kubeClient *kubernetes.Clientset, restConfig *rest.Config,
	snapshotOpts *snapshot.Options, log log.Logger, vClusterConfig *vclusterconfig.VirtualClusterConfig) error {
	tempPath := fmt.Sprintf("%s/vcluster-snapshot-%d.tar.gz", dataMountPath, time.Now().Unix())
	localPath := snapshotOpts.File.Path
	if !vCluster.IsStandalone {
		// For non-standalone, we need to write the snapshot to the syncer PVC first, then download it via exec.
		snapshotOpts.File.Path = tempPath
	}

	log.Infof("Creating snapshot request...")
	snapshotRequest, err := snapshot.CreateSnapshotRequestResources(
		ctx, vCluster.Namespace, vClusterConfig.Name, vClusterConfig, snapshotOpts, kubeClient)
	if err != nil {
		return fmt.Errorf("create snapshot request: %w", err)
	}

	log.Infof("Waiting for snapshot to complete...")
	if err := waitForSnapshotRequest(ctx, kubeClient, vCluster.Namespace, snapshotRequest.Name); err != nil {
		return err
	}

	if vCluster.IsStandalone {
		// The file backend writes with 0600 already; chmod is a no-op but kept for safety.
		_ = os.Chmod(localPath, 0600)
		log.Infof("Snapshot saved to %s", localPath)
		return nil
	}

	targetPod, err := findVClusterPod(vCluster)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(localPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("create local file %s: %w", localPath, err)
	}
	defer f.Close()

	log.Infof("Downloading snapshot from pod %s to %s...", targetPod.Name, localPath)
	if err := podhelper.ExecStream(ctx, restConfig, &podhelper.ExecStreamOptions{
		Pod:       targetPod.Name,
		Namespace: vCluster.Namespace,
		Container: "syncer",
		Command:   []string{"cat", tempPath},
		Stdout:    f,
		Stderr:    os.Stderr,
	}); err != nil {
		_ = os.Remove(localPath)
		return fmt.Errorf("download snapshot from pod: %w", err)
	}

	if _, _, err := podhelper.ExecBuffered(ctx, restConfig, vCluster.Namespace,
		targetPod.Name, "syncer", []string{"rm", "-f", tempPath}, nil); err != nil {
		log.Warnf("Failed to remove temp snapshot file %s from pod: %v", tempPath, err)
	}

	log.Infof("Snapshot saved to %s", localPath)
	return nil
}

func restoreFromLocalFile(ctx context.Context, vCluster *find.VCluster,
	kubeClient *kubernetes.Clientset, restConfig *rest.Config,
	snapshotOpts *snapshot.Options, podOpts *pod.Options,
	log log.Logger, cmdArgs []string) error {
	tempPath := fmt.Sprintf("%s/vcluster-restore-%d.tar.gz", dataMountPath, time.Now().Unix())
	localPath := snapshotOpts.File.Path
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot file not found: %s", localPath)
	}

	if vCluster.IsStandalone {
		// For standalone, we can read directly from the local file instead of going through the syncer PVC.
		return restoreStandaloneVCluster(snapshotOpts, cmdArgs, log)
	}

	// For non-standalone, we need to upload the local file into the syncer container first, then point the restore command at it.
	snapshotOpts.File.Path = tempPath

	// Stream the local file into the syncer PVC via exec before pausing.
	// The PVC (and the staged file) persist through scale-to-zero.
	targetPod, err := findVClusterPod(vCluster)
	if err != nil {
		return err
	}

	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local snapshot %s: %w", localPath, err)
	}
	defer f.Close()

	log.Infof("Uploading %s to pod %s at %s...", localPath, targetPod.Name, tempPath)
	if err := podhelper.ExecStream(ctx, restConfig, &podhelper.ExecStreamOptions{
		Pod:       targetPod.Name,
		Namespace: vCluster.Namespace,
		Container: "syncer",
		Command:   []string{"/bin/sh", "-c", "cat > " + tempPath},
		Stdin:     f,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
	}); err != nil {
		return fmt.Errorf("upload snapshot to pod: %w", err)
	}

	return runRestorePod(ctx, kubeClient, restConfig, vCluster, snapshotOpts, podOpts, log, cmdArgs)
}

func waitForSnapshotRequest(ctx context.Context, kubeClient *kubernetes.Clientset,
	namespace, name string) error {
	return wait.PollUntilContextTimeout(ctx, 5*time.Second, 30*time.Minute, true,
		func(ctx context.Context) (bool, error) {
			cm, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return false, fmt.Errorf("get snapshot request ConfigMap: %w", err)
			}
			req, err := snapshot.UnmarshalSnapshotRequest(cm)
			if err != nil {
				return false, fmt.Errorf("unmarshal snapshot request: %w", err)
			}
			if req.Done() {
				if req.Status.Phase == snapshot.RequestPhaseCompleted {
					return true, nil
				}
				return false, fmt.Errorf("snapshot %s: %s", req.Status.Phase, req.Status.Error.Message)
			}
			return false, nil
		})
}

func findVClusterPod(vCluster *find.VCluster) (*corev1.Pod, error) {
	for i := range vCluster.Pods {
		p := &vCluster.Pods[i]
		if (vCluster.StatefulSet != nil || vCluster.Deployment != nil) && len(p.Name) > 0 {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no running pod found for vCluster %s", vCluster.Name)
}
