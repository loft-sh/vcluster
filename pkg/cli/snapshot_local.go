package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/loft-sh/log"
	vclusterconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/snapshot/container"
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

	localPath := snapshotOpts.LocalPath
	tempPath := fmt.Sprintf("%s/vcluster-snapshot-%d.tar.gz", dataMountPath, time.Now().Unix())

	// Translate to container:// for the in-cluster controller.
	containerOpts := *snapshotOpts
	containerOpts.Type = "container"
	containerOpts.Container = container.Options{Path: tempPath}

	log.Infof("Creating snapshot request...")
	snapshotRequest, err := snapshot.CreateSnapshotRequestResources(
		ctx, vCluster.Namespace, vClusterConfig.Name, vClusterConfig, &containerOpts, kubeClient)
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
	restoreVolumes bool, log log.Logger) error {

	localPath := snapshotOpts.LocalPath
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot file not found: %s", localPath)
	}

	containerOpts := *snapshotOpts
	containerOpts.Type = "container"

	cmdArgs := []string{"restore"}
	if restoreVolumes {
		cmdArgs = append(cmdArgs, "--restore-volumes")
	}

	if vCluster.IsStandalone {
		containerOpts.Container = container.Options{Path: localPath}
		return restoreStandaloneVCluster(ctx, vCluster, &containerOpts, cmdArgs, log)
	}

	// For non-standalone, we need to upload the local file into the syncer container first, then point the restore command at it.
	tempPath := fmt.Sprintf("%s/vcluster-restore-%d.tar.gz", dataMountPath, time.Now().Unix())
	containerOpts.Container = container.Options{Path: tempPath}

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

	log.Infof("Pausing vCluster %s", vCluster.Name)
	if err := pauseVCluster(ctx, kubeClient, vCluster, log); err != nil {
		return fmt.Errorf("pause vCluster %s: %w", vCluster.Name, err)
	}
	defer func() {
		log.Infof("Resuming vCluster %s", vCluster.Name)
		if err := lifecycle.ResumeVCluster(ctx, kubeClient, vCluster.Name, vCluster.Namespace, true, log); err != nil {
			log.Warnf("Error resuming vCluster %s: %v", vCluster.Name, err)
		}
	}()

	command := append([]string{"/vcluster"}, cmdArgs...)
	return pod.RunSnapshotPod(ctx, restConfig, kubeClient, command, vCluster, podOpts, &containerOpts, log)
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
