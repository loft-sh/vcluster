package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/blang/semver/v4"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/lifecycle"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/snapshot/file"
	"github.com/loft-sh/vcluster/pkg/snapshot/s3"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
)

type RestoreCmd struct {
	*flags.GlobalFlags

	S3   s3.Options
	File file.Options

	Storage string

	Log log.Logger
}

// NewRestore creates a new command
func NewRestore(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &RestoreCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "restore" + util.VClusterNameOnlyUseLine,
		Short: "Restores a virtual cluster from snapshot",
		Long: `#######################################################
################# vcluster restore ####################
#######################################################
Restore a virtual cluster.

Example:
vcluster restore test --namespace test
#######################################################
	`,
		Args:              util.VClusterNameOnlyValidator,
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Storage, "storage", "s3", "The storage to restore from. Can be either s3 or file")

	// add storage flags
	file.AddFileFlags(cobraCmd.Flags(), &cmd.File)
	s3.AddS3Flags(cobraCmd.Flags(), &cmd.S3)
	return cobraCmd
}

func (cmd *RestoreCmd) Run(ctx context.Context, args []string) error {
	vClusterName := args[0]
	vCluster, err := find.GetVCluster(ctx, globalFlags.Context, vClusterName, globalFlags.Namespace, cmd.Log)
	if err != nil {
		return err
	}

	// build kubernetes client
	restClient, err := vCluster.ClientFactory.ClientConfig()
	if err != nil {
		return err
	}
	kubeClient, err := kubernetes.NewForConfig(restClient)
	if err != nil {
		return err
	}

	// check if snapshot is supported
	if vCluster.Version != "dev-next" {
		version, err := semver.Parse(strings.TrimPrefix(vCluster.Version, "v"))
		if err != nil {
			return fmt.Errorf("parsing vCluster version: %w", err)
		}

		// check if version matches
		if version.LT(semver.MustParse(minSnapshotVersion)) {
			return fmt.Errorf("vCluster version %s snapshotting is not supported", vCluster.Version)
		}
	}

	// pause vCluster
	cmd.Log.Infof("Pausing vCluster %s", vCluster.Name)
	err = cli.PauseVCluster(ctx, kubeClient, vCluster, cmd.Log)
	if err != nil {
		return fmt.Errorf("pause vCluster %s: %w", vCluster.Name, err)
	}

	// try to scale up the vCluster again
	defer func() {
		cmd.Log.Infof("Resuming vCluster %s after it was paused", vCluster.Name)
		err = lifecycle.ResumeVCluster(ctx, kubeClient, vCluster.Name, vCluster.Namespace, cmd.Log)
		if err != nil {
			cmd.Log.Warnf("Error resuming vCluster %s: %v", vCluster.Name, err)
		}
	}()

	// now restore vCluster
	err = cmd.restoreVCluster(ctx, kubeClient, vCluster)
	if err != nil {
		return fmt.Errorf("restore vCluster %s: %w", vCluster.Name, err)
	}

	return nil
}

func (cmd *RestoreCmd) restoreVCluster(ctx context.Context, kubeClient *kubernetes.Clientset, vCluster *find.VCluster) error {
	// get pod spec
	var podSpec *corev1.PodSpec
	if vCluster.StatefulSet != nil {
		podSpec = &vCluster.StatefulSet.Spec.Template.Spec
	} else if vCluster.Deployment != nil {
		podSpec = &vCluster.Deployment.Spec.Template.Spec
	} else {
		return fmt.Errorf("vCluster %s has no StatefulSet or Deployment", vCluster.Name)
	}

	// now start the snapshot pod that takes the snapshot
	restorePod, err := cmd.startRestorePod(ctx, kubeClient, vCluster.Namespace, vCluster.Name, podSpec)
	if err != nil {
		return fmt.Errorf("starting snapshot pod: %w", err)
	}

	// create interrupt channel
	sigint := make(chan os.Signal, 1)
	defer func() {
		// make sure we won't interfere with interrupts anymore
		signal.Stop(sigint)

		// delete the restore pod when we are done
		_ = kubeClient.CoreV1().Pods(restorePod.Namespace).Delete(ctx, restorePod.Name, metav1.DeleteOptions{})
	}()

	// also delete on interrupt
	go func() {
		// interrupt signal sent from terminal
		signal.Notify(sigint, os.Interrupt)
		// sigterm signal sent from kubernetes
		signal.Notify(sigint, syscall.SIGTERM)

		// wait until we get killed
		<-sigint

		// cleanup virtual cluster
		err = kubeClient.CoreV1().Pods(restorePod.Namespace).Delete(ctx, restorePod.Name, metav1.DeleteOptions{
			GracePeriodSeconds: ptr.To(int64(1)),
		})
		if err != nil {
			klog.Warningf("Error deleting snapshot pod: %v", err)
		}
		os.Exit(1)
	}()

	// wait for pod to become ready
	err = waitForReadyPod(ctx, kubeClient, restorePod.Namespace, restorePod.Name, "restore", cmd.Log)
	if err != nil {
		return fmt.Errorf("waiting for restore pod to become ready: %w", err)
	}

	// now log the snapshot pod
	reader, err := kubeClient.CoreV1().Pods(restorePod.Namespace).GetLogs(restorePod.Name, &corev1.PodLogOptions{
		Follow: true,
	}).Stream(ctx)
	if err != nil {
		return fmt.Errorf("stream restore pod logs: %w", err)
	}
	defer reader.Close()

	// stream into stdout
	cmd.Log.Infof("Printing logs of restore pod...")
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return fmt.Errorf("write restore pod logs: %w", err)
	}

	// check restore pod for exit code
	exitCode, err := waitForCompletedPod(ctx, kubeClient, restorePod.Namespace, restorePod.Name, "restore", cmd.Log)
	if err != nil {
		return err
	}

	// check exit code of restore pod
	if exitCode != 0 {
		return fmt.Errorf("restore pod failed: exit code %d", exitCode)
	}

	return nil
}

func (cmd *RestoreCmd) startRestorePod(ctx context.Context, kubeClient *kubernetes.Clientset, namespace, vCluster string, podSpec *corev1.PodSpec) (*corev1.Pod, error) {
	var syncerContainer *corev1.Container
	for _, container := range podSpec.Containers {
		if container.Name == "syncer" {
			syncerContainer = &container
			break
		}
	}
	if syncerContainer == nil {
		return nil, fmt.Errorf("couldn't find syncer container")
	}

	// build args
	env := syncerContainer.Env
	options, err := toOptionsString(&snapshot.Options{
		S3:   cmd.S3,
		File: cmd.File,
	})
	if err != nil {
		return nil, err
	}
	env = append(env, corev1.EnvVar{
		Name:  "VCLUSTER_STORAGE_OPTIONS",
		Value: options,
	})

	// build the pod spec
	newPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "vcluster-restore-",
			Namespace:    namespace,
			Labels: map[string]string{
				"app": "vcluster-restore",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy:                 corev1.RestartPolicyNever,
			ServiceAccountName:            podSpec.ServiceAccountName,
			TerminationGracePeriodSeconds: ptr.To(int64(1)),
			NodeSelector:                  podSpec.NodeSelector,
			Affinity:                      podSpec.Affinity,
			Tolerations:                   podSpec.Tolerations,
			SecurityContext:               podSpec.SecurityContext,
			Volumes:                       podSpec.Volumes,
			Containers: []corev1.Container{
				{
					Name:            "restore",
					Image:           syncerContainer.Image,
					Command:         []string{"/vcluster", "restore", "--storage", cmd.Storage},
					SecurityContext: syncerContainer.SecurityContext,
					Env:             env,
					EnvFrom:         syncerContainer.EnvFrom,
					VolumeMounts:    syncerContainer.VolumeMounts,
				},
			},
		},
	}

	// add persistent volume claim volume if necessary
	for _, volumeMount := range syncerContainer.VolumeMounts {
		if volumeMount.Name == "data" {
			// check if its part of the pod spec
			found := false
			for _, volume := range newPod.Spec.Volumes {
				if volume.Name == volumeMount.Name {
					found = true
					break
				}
			}
			if !found {
				newPod.Spec.Volumes = append(newPod.Spec.Volumes, corev1.Volume{
					Name: volumeMount.Name,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "data-" + vCluster + "-0",
						},
					},
				})
			}
		}
	}

	// create the pod
	cmd.Log.Infof("Starting restore pod for vCluster %s/%s...", namespace, vCluster)
	newPod, err = kubeClient.CoreV1().Pods(namespace).Create(ctx, newPod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating restore pod: %w", err)
	}

	return newPod, nil
}
