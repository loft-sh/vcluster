package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	vclusterconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/snapshot/pod"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	apinet "k8s.io/apimachinery/pkg/util/net"
)

const (
	RestoreResourceQuota = "vcluster-restore"
)

func Restore(ctx context.Context, args []string, globalFlags *flags.GlobalFlags, snapshot *snapshot.Options, pod *pod.Options, newVCluster, restoreVolumes bool, log log.Logger) error {
	// init kube client and vCluster
	vCluster, kubeClient, restConfig, err := initSnapshotCommand(ctx, args, globalFlags, snapshot, log)
	if err != nil {
		return err
	}

	return restoreVCluster(ctx, kubeClient, restConfig, vCluster, snapshot, pod, newVCluster, restoreVolumes, log)
}

func restoreVCluster(ctx context.Context, kubeClient *kubernetes.Clientset, restConfig *rest.Config, vCluster *find.VCluster, snapshotOptions *snapshot.Options, podOptions *pod.Options, newVCluster bool, restoreVolumes bool, log log.Logger) error {
	// build command arguments
	cmdArgs := []string{"restore"}
	if newVCluster {
		cmdArgs = append(cmdArgs, "--new-vcluster")
	}
	if restoreVolumes {
		cmdArgs = append(cmdArgs, "--restore-volumes")
	}

	if vCluster.IsStandalone {
		isStandaloneControlPlane, err := CheckStandaloneControlPlane()
		if err != nil {
			return fmt.Errorf("failed to check if current environment is the standalone control plane: %w", err)
		}

		if !isStandaloneControlPlane {
			return fmt.Errorf("could not find standalone control plane on the current host")
		}

		log.Infof("Stopping vCluster system service")
		err = exec.Command("systemctl", "stop", "vcluster").Run()
		if err != nil {
			return fmt.Errorf("failed to stop vcluster: %w", err)
		}

		defer func() {
			log.Infof("Resuming vCluster system service after restore")
			err := exec.Command("systemctl", "start", "vcluster").Run()
			if err != nil {
				log.Warnf("failed to start vcluster: %v", err)
			}
		}()

		return runRestoreCommand(vCluster, snapshotOptions, cmdArgs)
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

	return pod.RunSnapshotPod(ctx, restConfig, kubeClient, command, vCluster, podOptions, snapshotOptions, log)
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

// CheckStandaloneControlPlane checks if the current host has the standalone control plane installed
// by verifying the presence of a Linux service named "vcluster".
func CheckStandaloneControlPlane() (bool, error) {
	if runtime.GOOS != "linux" {
		return false, fmt.Errorf("not supported on %s", runtime.GOOS)
	}

	// Check if the vcluster service is active using systemctl
	cmdOutput, err := exec.Command("systemctl", "is-active", "vcluster").Output()
	if err != nil {
		return false, fmt.Errorf("failed to check if vcluster service is active: %w", err)
	}

	return strings.TrimSpace(string(cmdOutput)) == "active", nil
}

func runRestoreCommand(vCluster *find.VCluster, snapshotOptions *snapshot.Options, args []string) error {
	// parse vcluster config
	vClusterConfig, err := vclusterconfig.ParseConfig(constants.StandaloneDefaultConfigPath, vCluster.Name, nil)
	if err != nil {
		return fmt.Errorf("failed to parse vcluster config: %w", err)
	}

	restoreCmd := exec.Command(filepath.Join(vClusterConfig.ControlPlane.Standalone.DataDir, "bin", "vcluster"), args...)

	// convert snapshot options to string
	optionsString, err := pod.ToOptionsString(snapshotOptions)
	if err != nil {
		return fmt.Errorf("failed to convert snapshot options to string: %w", err)
	}

	// fetch host IP from network interface
	hostIP, err := apinet.ChooseHostInterface()
	if err != nil {
		return err
	}

	// set environment variables
	restoreCmd.Env = os.Environ()
	restoreCmd.Env = append(restoreCmd.Env, "VCLUSTER_NAME="+vCluster.Name)
	restoreCmd.Env = append(restoreCmd.Env, "VCLUSTER_STORAGE_OPTIONS="+optionsString)
	restoreCmd.Env = append(restoreCmd.Env, "VCLUSTER_STANDALONE_IP_ADDRESS="+hostIP.String())

	// set stdout and stderr
	restoreCmd.Stdout = os.Stdout
	restoreCmd.Stderr = os.Stderr

	err = restoreCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to restore vcluster: %w", err)
	}

	return nil
}
