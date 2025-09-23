package certs

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/blang/semver"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type RotationCmd string

const (
	RotationCmdCerts   RotationCmd = "rotate"
	RotationCmdCACerts RotationCmd = "rotate-ca"
)

const minVersion = "0.27.0-alpha.9"

// Rotate triggers the rotate commands in the backend.
// Depending on if the virtual cluster has persistence it either:
// - Pauses the current virtual cluster, spawns an extra pod, executes the rotation and resumes the virtual cluster.
// - Executes the rotation directly in the currently running syncer pod.
func Rotate(ctx context.Context, vClusterName string, rotationCmd RotationCmd, globalFlags *flags.GlobalFlags, log log.Logger) error {
	vCluster, err := find.GetVCluster(ctx, globalFlags.Context, vClusterName, globalFlags.Namespace, log)
	if err != nil {
		return fmt.Errorf("finding virtual cluster: %w", err)
	}

	// check if rotate is supported
	version, err := semver.Parse(strings.TrimPrefix(vCluster.Version, "v"))
	if err == nil {
		// only check if version matches if vCluster actually has a parsable version
		if version.LT(semver.MustParse(minVersion)) {
			return fmt.Errorf("cert rotation is not supported in vCluster version %s", vCluster.Version)
		}
	}

	// abort in case the virtual cluster has a non-running status.
	if vCluster.Status != find.StatusRunning {
		return fmt.Errorf("aborting operation because virtual cluster %q has status %q", vCluster.Name, vCluster.Status)
	}

	kubeConfig, err := vCluster.ClientFactory.ClientConfig()
	if err != nil {
		return fmt.Errorf("getting client config: %w", err)
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return fmt.Errorf("creating kubernetes client: %w", err)
	}

	var cmd string
	dev, validityPeriod := os.Getenv("DEVELOPMENT"), os.Getenv("VCLUSTER_CERTS_VALIDITYPERIOD")
	if dev == "true" && validityPeriod != "" {
		cmd = fmt.Sprintf("DEVELOPMENT=true VCLUSTER_CERTS_VALIDITYPERIOD=%s ", validityPeriod)
	}

	switch rotationCmd {
	case RotationCmdCerts:
		cmd = fmt.Sprintf("%s/vcluster certs %s", cmd, RotationCmdCerts)
	case RotationCmdCACerts:
		cmd = fmt.Sprintf("%s/vcluster certs %s", cmd, RotationCmdCACerts)
	default:
		return fmt.Errorf("unknown rotation command: %s", rotationCmd)
	}

	return execRotate(ctx, "certs-rotate", cmd, kubeClient, vCluster, log)
}

func execRotate(ctx context.Context, containerName, cmd string, kubeClient *kubernetes.Clientset, vCluster *find.VCluster, log log.Logger) error {
	// TODO(johannesfrey): For standalone the flow would need to be something like:
	// - systemctl stop vcluster.service
	// - /var/lib/vcluster/bin/vcluster certs rotate
	// - systemctl start vcluster.service

	pvc, err := usesPVC(vCluster)
	if err != nil {
		return fmt.Errorf("checking for persistence: %w", err)
	}

	// If the vCluster has persistence we have to pause it in order to be able to mount
	// the data dir to the extra pod.
	if pvc {
		log.Infof("Pausing vCluster %s", vCluster.Name)
		if err := lifecycle.PauseVCluster(ctx, kubeClient, vCluster.Name, vCluster.Namespace, true, log); err != nil {
			return err
		}

		log.Infof("Running %s pod", containerName)
		err = podhelper.RunSyncerPod(ctx, containerName, kubeClient, []string{"sh", "-c", cmd}, vCluster, nil, log)
		if err != nil {
			return fmt.Errorf("running %s pod: %w", containerName, err)
		}

		log.Infof("Resuming vCluster %s after it was paused", vCluster.Name)
		if err := lifecycle.ResumeVCluster(ctx, kubeClient, vCluster.Name, vCluster.Namespace, true, log); err != nil {
			return fmt.Errorf("resuming virtual cluster %s: %w", vCluster.Name, err)
		}

		// Won't do anything in case deployed etcd does not exist.
		return lifecycle.DeletePods(ctx, kubeClient, "app=vcluster-etcd,release="+vCluster.Name, vCluster.Namespace)
	}

	if len(vCluster.Pods) == 0 {
		return fmt.Errorf("no target pod found in vCluster %s", vCluster.Name)
	}

	kubeConfig, err := vCluster.ClientFactory.ClientConfig()
	if err != nil {
		return err
	}

	log.Info("Executing in syncer pod")
	err = podhelper.ExecStream(ctx, kubeConfig, &podhelper.ExecStreamOptions{
		Pod:       vCluster.Pods[0].Name,
		Namespace: vCluster.Namespace,
		Container: "syncer",
		Command:   []string{"sh", "-c", cmd},
		Stdout:    os.Stdout,
		Stderr:    os.Stdout,
	})
	if err != nil {
		return fmt.Errorf("executing command in syncer pod: %w", err)
	}

	if err := lifecycle.DeletePods(ctx, kubeClient, "app=vcluster,release="+vCluster.Name, vCluster.Namespace); err != nil {
		return fmt.Errorf("deleting pod of virtual cluster %s: %w", vCluster.Name, err)
	}

	// Won't do anything in case deployed etcd does not exist.
	return lifecycle.DeletePods(ctx, kubeClient, "app=vcluster-etcd,release="+vCluster.Name, vCluster.Namespace)
}

func usesPVC(vCluster *find.VCluster) (bool, error) {
	var podSpec *corev1.PodSpec
	if vCluster.StatefulSet != nil {
		podSpec = &vCluster.StatefulSet.Spec.Template.Spec
	} else if vCluster.Deployment != nil {
		podSpec = &vCluster.Deployment.Spec.Template.Spec
	} else {
		return false, fmt.Errorf("vCluster %s has no StatefulSet or Deployment", vCluster.Name)
	}

	var syncerContainer *corev1.Container
	for _, container := range podSpec.Containers {
		if container.Name == "syncer" {
			syncerContainer = &container
			break
		}
	}
	if syncerContainer == nil {
		return false, fmt.Errorf("couldn't find syncer container")
	}

	for _, volumeMount := range syncerContainer.VolumeMounts {
		if volumeMount.Name == "data" {
			if vCluster.StatefulSet != nil {
				for _, vct := range vCluster.StatefulSet.Spec.VolumeClaimTemplates {
					if vct.Name == volumeMount.Name {
						return true, nil
					}
				}
			}
			for _, volume := range podSpec.Volumes {
				if volume.Name == volumeMount.Name && volume.PersistentVolumeClaim != nil {
					return true, nil
				}
			}
		}
	}

	return false, nil
}
