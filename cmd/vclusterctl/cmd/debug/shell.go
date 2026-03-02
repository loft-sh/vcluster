package debug

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	debugshellutil "github.com/loft-sh/vcluster/pkg/util/debugshell"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type DebugCmd struct {
	Log log.Logger
	*flags.GlobalFlags

	VClusterPodName string
	Target          string
	Command         []string
}

func NewShellCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &DebugCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "shell",
		Short: "shell VCLUSTER_NAME [flags] -- [COMMAND] [args...] Debug a virtual cluster using ephemeral containers",
		Long: `#########################################################################
#################### vcluster debug ###################################
#########################################################################
Debug a virtual cluster by running commands in an ephemeral container

Example:
vcluster debug shell my-vcluster --target=syncer
vcluster debug shell my-vcluster --pod=my-vcluster-pod-0 --target=syncer
#########################################################################
`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				cmd.Command = args[1:]
			}
			return cmd.Run(cobraCmd.Context(), args)
		},
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
	}

	cobraCmd.Flags().StringVar(&cmd.VClusterPodName, "pod", "", "vCluster pod name to create a shell for")
	cobraCmd.Flags().StringVar(&cmd.Target, "target", "syncer", "Target container to debug")

	return cobraCmd
}

func (cmd *DebugCmd) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("please specify a virtual cluster name")
	}
	vClusterName := args[0]
	// Resolve target vCluster and pick the pod to attach an ephemeral debug container to.
	client, err := find.CreateKubeClient(cmd.Context)
	if err != nil {
		return fmt.Errorf("cannot create Kubernetes client: %w", err)
	}

	virtualCluster, err := find.GetVCluster(ctx, cmd.Context, vClusterName, cmd.Namespace, cmd.Log)
	if err != nil {
		return fmt.Errorf("cannot find virtual cluster: %w", err)
	}

	pod, err := debugshellutil.SelectTargetPod(virtualCluster.Pods, cmd.VClusterPodName)
	if err != nil {
		return fmt.Errorf("cannot find vCluster pod: %w", err)
	}
	debuggerNamePrefix := fmt.Sprintf("debug-shell-%s", cmd.Target)

	// Reuse existing shell to avoid duplicates; only create when none exist.
	if debuggerName, running := debugshellutil.FindExistingDebugShell(pod, debuggerNamePrefix); debuggerName != "" {
		if !running {
			err = cmd.waitForContainer(ctx, client, pod.Namespace, pod.Name, debuggerName)
			if err != nil {
				return fmt.Errorf("error waiting for debug container: %w", err)
			}
		}
		return executeCommand(debuggerName, pod.Name, pod.Namespace, cmd.Command, cmd.Log)
	}

	targetContainer := debugshellutil.FindTargetContainer(pod, cmd.Target)
	if targetContainer == nil {
		return fmt.Errorf("could not find container %q in pod %q", cmd.Target, pod.Name)
	}

	// Build env for the debug container, preserving target container envs.
	envs := append(targetContainer.Env, debugshellutil.BuildEnvs(pod.Name, virtualCluster.Version)...)
	if virtualCluster.VirtualClusterInstance != nil && virtualCluster.VirtualClusterInstance.Status.VirtualCluster != nil {
		embeddedEtcdEnabled, err := debugshellutil.IsEmbeddedEtcdEnabled([]byte(virtualCluster.VirtualClusterInstance.Status.VirtualCluster.HelmRelease.Values))
		if err != nil {
			return fmt.Errorf("cannot determine whether embedded etcd is enabled: %w", err)
		}
		if embeddedEtcdEnabled {
			envs = debugshellutil.AppendEtcdEnvs(envs, debugshellutil.ShellBannerEnv, vClusterName, len(virtualCluster.Pods))
		}
	}
	// Create debug container name
	debuggerName := fmt.Sprintf("%s-%s", debuggerNamePrefix, random.String(5))
	debugContainer := debugshellutil.CreateEphemeralContainer(
		targetContainer, envs, debuggerName, debugshellutil.ShellBannerEnv, debugshellutil.DefaultTTLForEphemeralContainer)

	ephemeralContainers := []corev1.EphemeralContainer{debugContainer}
	if len(pod.Spec.EphemeralContainers) > 0 {
		ephemeralContainers = append(ephemeralContainers, pod.Spec.EphemeralContainers...)
	}

	// Add ephemeral container to pod
	pod, err = client.CoreV1().Pods(pod.Namespace).UpdateEphemeralContainers(ctx, pod.Name, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
		Spec: corev1.PodSpec{
			EphemeralContainers: ephemeralContainers,
		},
	}, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error creating ephemeral container: %w", err)
	}
	cmd.Log.Infof("Debugger %s created for container %s in pod %s", debuggerName, cmd.Target, pod.Name)

	// Wait for the ephemeral container to be ready before exec.
	err = cmd.waitForContainer(ctx, client, pod.Namespace, pod.Name, debuggerName)
	if err != nil {
		return fmt.Errorf("error waiting for debug container: %w", err)
	}

	// Exec into the debug container and start a shell.
	return executeCommand(debuggerName, pod.Name, pod.Namespace, cmd.Command, cmd.Log)
}

func (cmd *DebugCmd) waitForContainer(ctx context.Context, client kubernetes.Interface, namespace, podName, containerName string) error {
	// Poll until the ephemeral container is running (or terminated).
	return wait.PollUntilContextTimeout(ctx, time.Second, time.Minute*2, true, func(ctx context.Context) (bool, error) {
		pod, err := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, container := range pod.Status.EphemeralContainerStatuses {
			if container.Name == containerName {
				if container.State.Running != nil {
					return true, nil
				}
				if container.State.Terminated != nil {
					return false, fmt.Errorf("ephemeral container %s terminated: %s", containerName, container.State.Terminated.Message)
				}
				// Container is still in waiting state
				cmd.Log.Infof("Waiting for ephemeral container %s to start...", containerName)
				return false, nil
			}
		}

		return false, nil
	})
}

func executeCommand(containerName, pod, namespace string, command []string, log log.Logger) error {
	// Delegate to kubectl exec to get an interactive shell in the ephemeral container.
	args := []string{"exec", "-n", namespace, pod, "-it", "-c", containerName, "--"}
	if len(command) > 0 {
		args = append(args, command...)
	} else {
		args = append(args, "/bin/sh", "-c", "echo \"$"+debugshellutil.ShellBannerEnv+"\" && ash")
	}
	execCmd := exec.Command("kubectl", args...)

	execCmd.Env = os.Environ()
	execCmd.Stdout = os.Stdout
	execCmd.Stdin = os.Stdin
	execCmd.Stderr = os.Stderr
	err := execCmd.Start()
	if err != nil {
		return err
	}
	err = execCmd.Wait()
	if exitError, ok := lo.ErrorsAs[*exec.ExitError](err); ok {
		log.Errorf("Error executing command: %v", err)
		os.Exit(exitError.ExitCode())
	}

	return err
}
