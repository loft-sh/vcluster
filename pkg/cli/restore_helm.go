package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	vclusterconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/snapshot/pod"
	standaloneutil "github.com/loft-sh/vcluster/pkg/util/standalone"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	RestoreResourceQuota = "vcluster-restore"
)

func Restore(ctx context.Context, args []string, globalFlags *flags.GlobalFlags, snapshotOpts *snapshot.Options, podOpts *pod.Options, newVCluster, restoreVolumes, standalone bool, log log.Logger) error {
	// init kube client and vCluster
	vCluster, kubeClient, restConfig, err := initSnapshotCommand(ctx, args, globalFlags, snapshotOpts, log, true, standalone)
	if err != nil {
		return err
	}

	return restoreVCluster(ctx, kubeClient, restConfig, vCluster, snapshotOpts, podOpts, newVCluster, restoreVolumes, log)
}

func restoreVCluster(ctx context.Context, kubeClient *kubernetes.Clientset, restConfig *rest.Config, vCluster *find.VCluster, snapshotOpts *snapshot.Options, podOptions *pod.Options, newVCluster bool, restoreVolumes bool, log log.Logger) error {
	cmdArgs := []string{"restore"}
	if newVCluster {
		cmdArgs = append(cmdArgs, "--new-vcluster")
	}
	if restoreVolumes {
		cmdArgs = append(cmdArgs, "--restore-volumes")
	}

	if vCluster.IsStandalone {
		return restoreStandaloneVCluster(ctx, vCluster, snapshotOpts, cmdArgs, log)
	}

	// pause vCluster
	log.Infof("Pausing vCluster %s", vCluster.Name)
	err := pauseVCluster(ctx, kubeClient, vCluster, log)
	if err != nil {
		return fmt.Errorf("pause vCluster %s: %w", vCluster.Name, err)
	}

	// try to scale up the vCluster again
	defer func() {
		log.Infof("Resuming vCluster %s after it was paused", vCluster.Name)
		err = lifecycle.ResumeVCluster(ctx, kubeClient, vCluster.Name, vCluster.Namespace, true, log)
		if err != nil {
			log.Warnf("Error resuming vCluster %s: %v", vCluster.Name, err)
		}
	}()

	// set missing pod options and run snapshot restore pod
	command := append([]string{"/vcluster"}, cmdArgs...)
	return pod.RunSnapshotPod(ctx, restConfig, kubeClient, command, vCluster, podOptions, snapshotOpts, log)
}

// restoreStandaloneVCluster stops the standalone service, invokes the vcluster binary
// directly to perform the restore, and always attempts to start the service again
// before returning. If both the restore and restart fail, the returned error retains
// both failures. The CLI must run on the same host as the standalone installation
// because it needs filesystem access to the binary and config.
func restoreStandaloneVCluster(ctx context.Context, vCluster *find.VCluster, snapshotOpts *snapshot.Options, cmdArgs []string, log log.Logger) (retErr error) {
	vClusterConfig, err := vclusterconfig.LoadStandaloneConfig("", nil)
	if err != nil {
		return fmt.Errorf("load standalone config: %w", err)
	}

	sm, err := standaloneutil.NewServiceManager()
	if err != nil {
		return err
	}

	if err := pro.CheckStandaloneHA(ctx, vClusterConfig); err != nil {
		return err
	}

	log.Infof("Stopping vCluster service")
	if err := sm.Stop(); err != nil {
		return fmt.Errorf("stop vCluster service: %w", err)
	}

	defer func() {
		log.Infof("Starting vCluster service")
		if startErr := sm.Start(); startErr != nil {
			restartErr := fmt.Errorf("restart vCluster service: %w", startErr)
			if retErr != nil {
				retErr = errors.Join(retErr, restartErr)
			} else {
				retErr = restartErr
			}
		}
	}()

	if err := runRestoreBinary(vClusterConfig, snapshotOpts, cmdArgs); err != nil {
		return fmt.Errorf("restore standalone vCluster: %w", err)
	}
	return nil
}

func runRestoreBinary(vClusterConfig *vclusterconfig.VirtualClusterConfig, snapshotOpts *snapshot.Options, args []string) error {
	binaryPath := filepath.Join(vClusterConfig.ControlPlane.Standalone.DataDir, "bin", "vcluster")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// Fall back to the currently executing binary (e.g. during development or
		// non-standard installs where the binary is not in the data directory).
		self, err := os.Executable()
		if err != nil {
			return fmt.Errorf("binary not found at %s and cannot determine current executable: %w", binaryPath, err)
		}
		binaryPath = self
	}

	optionsString, err := pod.ToOptionsString(snapshotOpts)
	if err != nil {
		return fmt.Errorf("serialise snapshot options: %w", err)
	}

	env := append(os.Environ(),
		"VCLUSTER_NAME="+vClusterConfig.Name,
		constants.VClusterStandaloneEnvVar+"=true",
		constants.VClusterStorageOptionsEnv+"="+optionsString,
	)

	cmd := exec.Command(binaryPath, args...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func pauseVCluster(ctx context.Context, kubeClient *kubernetes.Clientset, vCluster *find.VCluster, log log.Logger) error {
	// pause the vCluster
	err := lifecycle.PauseVCluster(ctx, kubeClient, vCluster.Name, vCluster.Namespace, true, log)
	if err != nil {
		return err
	}

	// restart the workloads
	err = lifecycle.DeletePods(ctx, kubeClient, "vcluster.loft.sh/managed-by="+vCluster.Name, vCluster.Namespace)
	if err != nil {
		return fmt.Errorf("delete vcluster workloads: %w", err)
	}
	err = lifecycle.DeleteMultiNamespaceVClusterWorkloads(ctx, kubeClient, vCluster.Name, vCluster.Namespace, log)
	if err != nil {
		return fmt.Errorf("delete vcluster multinamespace workloads: %w", err)
	}

	// ensure there is only one pvc
	err = ensurePVCs(ctx, kubeClient, vCluster, log)
	if err != nil {
		return err
	}

	// delete restore resource quota if exists
	_, err = kubeClient.CoreV1().ResourceQuotas(vCluster.Namespace).Get(ctx, RestoreResourceQuota, metav1.GetOptions{})
	if err == nil {
		err = kubeClient.CoreV1().ResourceQuotas(vCluster.Namespace).Delete(ctx, RestoreResourceQuota, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("delete restore resource quota %s: %w", RestoreResourceQuota, err)
		}
	}

	return nil
}

func ensurePVCs(ctx context.Context, kubeClient *kubernetes.Clientset, vCluster *find.VCluster, log log.Logger) error {
	// if not a statefulset we don't care
	if vCluster.StatefulSet == nil || len(vCluster.StatefulSet.Spec.VolumeClaimTemplates) == 0 {
		return nil
	}

	// two things we need to check for:
	// 1. if there is more than 1 pvc (this is the case for embedded etcd) then we delete all pvc's except the first one
	// 2. if there is no pvc and the statefulset has a persistent volume claim template we create the pvc
	pvcList, err := kubeClient.CoreV1().PersistentVolumeClaims(vCluster.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=vcluster,release=%s", vCluster.Name),
	})
	if err != nil {
		return fmt.Errorf("list vcluster pvcs: %w", err)
	}

	// handle the two cases now
	if len(pvcList.Items) == 0 {
		// create the pvc
		log.Infof("No vCluster pvcs found in namespace %s, creating a new one...", vCluster.Namespace)
		dataVolume := vCluster.StatefulSet.Spec.VolumeClaimTemplates[0]
		_, err := kubeClient.CoreV1().PersistentVolumeClaims(vCluster.Namespace).Create(ctx, &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: dataVolume.Name + "-" + vCluster.StatefulSet.Name + "-0",
			},
			Spec: dataVolume.Spec,
		}, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create vcluster pvc: %w", err)
		}
	} else if len(pvcList.Items) > 1 {
		// delete the non -0 ones
		for _, pvc := range pvcList.Items {
			if strings.HasSuffix(pvc.Name, "-0") {
				continue
			}

			log.Infof("Deleting vCluster pvc %s/%s", pvc.Namespace, pvc.Name)
			err = kubeClient.CoreV1().PersistentVolumeClaims(vCluster.Namespace).Delete(ctx, pvc.Name, metav1.DeleteOptions{})
			if err != nil {
				return fmt.Errorf("delete vcluster pvc: %w", err)
			}
		}
	}

	return nil
}
