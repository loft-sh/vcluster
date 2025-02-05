package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/blang/semver/v4"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/snapshot/file"
	"github.com/loft-sh/vcluster/pkg/snapshot/s3"
	"github.com/loft-sh/vcluster/pkg/util/clihelper"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
)

var minSnapshotVersion = "0.23.0-alpha.8"

type SnapshotCmd struct {
	*flags.GlobalFlags

	S3   s3.Options
	File file.Options

	Storage string

	Log log.Logger
}

// NewSnapshot creates a new command
func NewSnapshot(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &SnapshotCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "snapshot" + util.VClusterNameOnlyUseLine,
		Short: "Snapshot a virtual cluster",
		Long: `#######################################################
################# vcluster snapshot ###################
#######################################################
Snapshot a virtual cluster.

Example:
vcluster snapshot test --namespace test
#######################################################
	`,
		Args:              util.VClusterNameOnlyValidator,
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Storage, "storage", "s3", "The storage to snapshot to. Can be either s3 or file")

	// add storage flags
	file.AddFileFlags(cobraCmd.Flags(), &cmd.File)
	s3.AddS3Flags(cobraCmd.Flags(), &cmd.S3)
	return cobraCmd
}

func (cmd *SnapshotCmd) Run(ctx context.Context, args []string) error {
	vClusterName := args[0]
	vCluster, err := find.GetVCluster(ctx, globalFlags.Context, vClusterName, globalFlags.Namespace, cmd.Log)
	if err != nil {
		return err
	}

	// we cannot snapshot a sleeping / paused vCluster
	if vCluster.IsSleeping() || vCluster.Status == find.StatusPaused {
		return fmt.Errorf("cannot take a snapshot of a sleeping vCluster")
	} else if len(vCluster.Pods) == 0 {
		return fmt.Errorf("couldn't find vCluster pod")
	}

	// if it's a statefulset then try to get the pod with the suffix -0
	var vClusterPod *corev1.Pod
	for _, p := range vCluster.Pods {
		if strings.HasSuffix(p.Name, "-0") {
			vClusterPod = &p
			break
		}

		controller := metav1.GetControllerOf(vClusterPod)
		if controller == nil || controller.Kind != "StatefulSet" {
			vClusterPod = &p
		}
	}
	if vClusterPod == nil {
		return fmt.Errorf("couldn't find vCluster pod")
	}

	// check if snapshot is supported
	if vCluster.Version != "dev-next" {
		version, err := semver.Parse(strings.TrimPrefix(vCluster.Version, "v"))
		if err != nil {
			return fmt.Errorf("parsing vCluster version %s: %w", vCluster.Version, err)
		}

		// check if version matches
		if version.LT(semver.MustParse(minSnapshotVersion)) {
			return fmt.Errorf("vCluster version %s snapshotting is not supported", vCluster.Version)
		}
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

	// now start the snapshot pod that takes the snapshot
	snapshotPod, err := cmd.startSnapshotPod(ctx, kubeClient, vClusterPod)
	if err != nil {
		return fmt.Errorf("starting snapshot pod: %w", err)
	}
	defer func() {
		// delete the snapshot pod when we are done
		_ = kubeClient.CoreV1().Pods(snapshotPod.Namespace).Delete(ctx, snapshotPod.Name, metav1.DeleteOptions{})
	}()

	// also delete on interrupt
	go func() {
		sigint := make(chan os.Signal, 1)
		// interrupt signal sent from terminal
		signal.Notify(sigint, os.Interrupt)
		// sigterm signal sent from kubernetes
		signal.Notify(sigint, syscall.SIGTERM)

		// wait until we get killed
		<-sigint

		// cleanup virtual cluster
		err = kubeClient.CoreV1().Pods(snapshotPod.Namespace).Delete(ctx, snapshotPod.Name, metav1.DeleteOptions{
			GracePeriodSeconds: ptr.To(int64(1)),
		})
		if err != nil {
			klog.Warningf("Error deleting snapshot pod: %v", err)
		}
		os.Exit(1)
	}()

	// wait for pod to become ready
	err = waitForReadyPod(ctx, kubeClient, snapshotPod.Namespace, snapshotPod.Name, "snapshot", cmd.Log)
	if err != nil {
		return fmt.Errorf("waiting for snapshot pod to become ready: %w", err)
	}

	// now log the snapshot pod
	reader, err := kubeClient.CoreV1().Pods(snapshotPod.Namespace).GetLogs(snapshotPod.Name, &corev1.PodLogOptions{
		Follow: true,
	}).Stream(ctx)
	if err != nil {
		return fmt.Errorf("stream snapshot pod logs: %w", err)
	}
	defer reader.Close()

	// stream into stdout
	cmd.Log.Infof("Printing logs of snapshot pod...")
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return fmt.Errorf("write snapshot pod logs: %w", err)
	}

	// check snapshot pod for exit code
	exitCode, err := waitForCompletedPod(ctx, kubeClient, snapshotPod.Namespace, snapshotPod.Name, "snapshot", cmd.Log)
	if err != nil {
		return err
	}

	// check exit code of snapshot container
	if exitCode != 0 {
		return fmt.Errorf("snapshot pod failed: exit code %d", exitCode)
	}

	return nil
}

func (cmd *SnapshotCmd) startSnapshotPod(ctx context.Context, kubeClient *kubernetes.Clientset, vClusterPod *corev1.Pod) (*corev1.Pod, error) {
	var syncerContainer *corev1.Container
	for _, container := range vClusterPod.Spec.Containers {
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
			GenerateName: "vcluster-snapshot-",
			Namespace:    vClusterPod.Namespace,
			Labels: map[string]string{
				"app": "vcluster-snapshot",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy:                 corev1.RestartPolicyNever,
			ServiceAccountName:            vClusterPod.Spec.ServiceAccountName,
			AutomountServiceAccountToken:  ptr.To(false),
			TerminationGracePeriodSeconds: ptr.To(int64(1)),
			NodeName:                      vClusterPod.Spec.NodeName,
			Tolerations:                   vClusterPod.Spec.Tolerations,
			SecurityContext:               vClusterPod.Spec.SecurityContext,
			Volumes:                       vClusterPod.Spec.Volumes,
			Containers: []corev1.Container{
				{
					Name:            "snapshot",
					Image:           syncerContainer.Image,
					Command:         []string{"/vcluster", "snapshot", "--storage", cmd.Storage},
					SecurityContext: syncerContainer.SecurityContext,
					Env:             env,
					EnvFrom:         syncerContainer.EnvFrom,
					VolumeMounts:    syncerContainer.VolumeMounts,
				},
			},
		},
	}

	// create the pod
	cmd.Log.Infof("Starting snapshot pod %s/%s...", vClusterPod.Namespace, vClusterPod.Name)
	newPod, err = kubeClient.CoreV1().Pods(vClusterPod.Namespace).Create(ctx, newPod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating snapshot pod: %w", err)
	}

	return newPod, nil
}

func getExitCode(pod *corev1.Pod, container string) int32 {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name != container {
			continue
		}

		if containerStatus.State.Terminated != nil {
			return containerStatus.State.Terminated.ExitCode
		}

		return -1
	}

	return -1
}

func waitForCompletedPod(ctx context.Context, kubeClient *kubernetes.Clientset, namespace, name, container string, log log.Logger) (int32, error) {
	exitCode := int32(-1)
	err := wait.PollUntilContextTimeout(ctx, time.Second*2, time.Minute, true, func(ctx context.Context) (bool, error) {
		pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			// this is a fatal
			return false, fmt.Errorf("error trying to retrieve pod %s/%s: %w", namespace, name, err)
		}

		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.Name != container {
				continue
			}

			if containerStatus.State.Running != nil {
				return false, nil
			} else if containerStatus.State.Terminated != nil {
				exitCode = containerStatus.State.Terminated.ExitCode
				return true, nil
			} else if containerStatus.State.Waiting != nil {
				if containerStatus.State.Waiting.Message != "" {
					return false, fmt.Errorf("error: %s container is waiting: %s (%s)", container, containerStatus.State.Waiting.Message, containerStatus.State.Waiting.Reason)
				} else if containerStatus.State.Waiting.Reason != "" {
					return false, fmt.Errorf("error: %s container is waiting: %s", container, containerStatus.State.Waiting.Reason)
				}

				return false, fmt.Errorf("error: %s container is waiting", container)
			}

			return false, nil
		}

		return false, nil
	})
	if err != nil {
		return exitCode, err
	}

	return exitCode, nil
}

func waitForReadyPod(ctx context.Context, kubeClient kubernetes.Interface, namespace, name, container string, log log.Logger) error {
	now := time.Now()
	err := wait.PollUntilContextTimeout(ctx, time.Second*2, time.Minute, true, func(ctx context.Context) (bool, error) {
		pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			// this is a fatal
			return false, fmt.Errorf("error trying to retrieve pod %s/%s: %w", namespace, name, err)
		}

		found := false
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.State.Running != nil && containerStatus.Ready {
				if containerStatus.Name == container {
					found = true
				}

				continue
			} else if containerStatus.State.Terminated != nil || (containerStatus.State.Waiting != nil && clihelper.CriticalStatus[containerStatus.State.Waiting.Reason]) {
				// if the container is completed that is fine as well
				if containerStatus.State.Terminated != nil && containerStatus.State.Terminated.ExitCode == 0 {
					found = true
					continue
				}

				reason := ""
				message := ""
				if containerStatus.State.Terminated != nil {
					reason = containerStatus.State.Terminated.Reason
					message = containerStatus.State.Terminated.Message
				} else if containerStatus.State.Waiting != nil {
					reason = containerStatus.State.Waiting.Reason
					message = containerStatus.State.Waiting.Message
				}

				out, err := kubeClient.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
					Container: container,
				}).Do(context.Background()).Raw()
				if err != nil {
					return false, fmt.Errorf("there seems to be an issue with pod %s/%s starting up: %s (%s)", namespace, name, message, reason)
				}

				return false, fmt.Errorf("there seems to be an issue with pod %s (%s - %s), logs:\n%s", name, message, reason, string(out))
			} else if containerStatus.State.Waiting != nil && time.Now().After(now.Add(time.Second*10)) {
				if containerStatus.State.Waiting.Message != "" {
					log.Infof("Please keep waiting, %s container is still starting up: %s (%s)", container, containerStatus.State.Waiting.Message, containerStatus.State.Waiting.Reason)
				} else if containerStatus.State.Waiting.Reason != "" {
					log.Infof("Please keep waiting, %s container is still starting up: %s", container, containerStatus.State.Waiting.Reason)
				} else {
					log.Infof("Please keep waiting, %s container is still starting up...", container)
				}

				now = time.Now()
			}

			return false, nil
		}

		return found, nil
	})
	if err != nil {
		return err
	}

	return nil
}

func toOptionsString(options *snapshot.Options) (string, error) {
	jsonBytes, err := json.Marshal(options)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(jsonBytes), nil
}
