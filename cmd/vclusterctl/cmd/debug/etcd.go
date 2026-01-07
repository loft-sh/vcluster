package debug

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/loft-sh/log"
	vclusterconfig "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/config"
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

	Target     string
	Profile    string
	KubeConfig string
	Command    []string
	errorChan  chan error
}

func NewDebugCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &DebugCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "etcd",
		Short: "etcd VCLUSTER_NAME [flags] -- [COMMAND] [args...]Debug a virtual cluster using ephemeral containers",
		Long: `#########################################################################
#################### vcluster debug ###################################
#########################################################################
Debug a virtual cluster by running commands in an ephemeral container

Example:
vcluster debug etcd my-vcluster --target=syncer
vcluster debug etcd my-vcluster --target=syncer --profile=general -- /bin/sh
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

	cobraCmd.Flags().StringVar(&cmd.Target, "target", "syncer", "Target container to debug")
	cobraCmd.Flags().StringVar(&cmd.Profile, "profile", "general", "Debug profile to use")
	cobraCmd.Flags().StringVar(&cmd.KubeConfig, "kubeconfig", "", "Path to kubeconfig")

	return cobraCmd
}

func (cmd *DebugCmd) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("please specify a virtual cluster name")
	}

	vClusterName := args[0]

	client, err := find.CreateKubeClient(cmd.Context)
	if err != nil {
		return fmt.Errorf("cannot create kube client: %w", err)
	}

	virtualCluster, err := find.GetVCluster(ctx, cmd.Context, vClusterName, cmd.Namespace, cmd.Log)
	if err != nil {
		return fmt.Errorf("cannot find virtual cluster %w", err)
	}

	if virtualCluster.VirtualClusterInstance.Status.VirtualCluster != nil {
		vConfig, err := config.ParseConfigBytes([]byte(virtualCluster.VirtualClusterInstance.Status.VirtualCluster.HelmRelease.Values), vClusterName, nil)
		if err != nil {
			return fmt.Errorf("cannot parse vcluster.yaml from virtualClusterInstance helm values %w", err)
		}
		if vConfig.BackingStoreType() != vclusterconfig.StoreTypeEmbeddedEtcd {
			return fmt.Errorf("your virtual cluster uses different type of storage than embedded etcd: %s", vConfig.BackingStoreType())
		}
		cmd.Log.Infof("virtual cluster %s uses embedded etcd as a storage", vClusterName)
	}

	if len(virtualCluster.Pods) == 0 {
		return fmt.Errorf("couldn't find vcluster pod in namespace %s", cmd.Namespace)
	}
	var endpoints string
	if len(virtualCluster.Pods) == 3 {
		endpoints = fmt.Sprintf(
			"https://localhost:2379,https://%s-0.%s-headless:2379,https://%s-1.%s-headless:2379,https://%s-2.%s-headless:2379",
			vClusterName, vClusterName, vClusterName, vClusterName, vClusterName, vClusterName,
		)
	} else {
		endpoints = "https://localhost:2379"
	}

	pod := &virtualCluster.Pods[0]

	// Get the target container's image
	var targetContainer *corev1.Container
	for _, container := range pod.Spec.Containers {
		if container.Name == cmd.Target {
			targetContainer = &container
			break
		}
	}
	if targetContainer == nil {
		return fmt.Errorf("couldn't find container %s in pod %s", cmd.Target, pod.Name)
	}

	// Create debug container name
	debuggerName := fmt.Sprintf("%s-debugger-"+random.String(5), cmd.Target)

	// set etcdctl env vars
	envs := targetContainer.Env

	etcdctlEnvs := []corev1.EnvVar{
		{
			Name:  "ETCDCTL_CACERT",
			Value: "/proc/1/root/data/pki/etcd/ca.crt",
		},
		{
			Name:  "ETCDCTL_KEY",
			Value: "/proc/1/root/data/pki/etcd/healthcheck-client.key",
		},
		{
			Name:  "ETCDCTL_CERT",
			Value: "/proc/1/root/data/pki/etcd/healthcheck-client.crt",
		},
		{
			Name:  "ETCDCTL_ENDPOINTS",
			Value: endpoints,
		},
		{
			Name:  "PATH",
			Value: "/usr/bin:/bin:/usr/local/bin:/proc/1/root/binaries",
		},
		{
			Name:  "DEBUG_MESSAGE",
			Value: ephemeralContainerDebugMessage(virtualCluster.Version),
		},
	}
	envs = append(envs, etcdctlEnvs...)

	// Configure the ephemeral container
	debugContainer := corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:                     debuggerName,
			Image:                    targetContainer.Image,
			Command:                  []string{"sleep", "inf"},
			Env:                      envs,
			TTY:                      true,
			Stdin:                    true,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			SecurityContext: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Add: []corev1.Capability{"SYS_PTRACE"},
				},
			},
		},
		TargetContainerName: cmd.Target,
	}

	ephemeralContainers := []corev1.EphemeralContainer{debugContainer}
	if len(pod.Spec.EphemeralContainers) > 0 {
		ephemeralContainers = append(ephemeralContainers, pod.Spec.EphemeralContainers...)
	}

	// Add ephemeral container to pod
	pod, err = client.CoreV1().Pods(cmd.Namespace).UpdateEphemeralContainers(ctx, pod.Name, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
		Spec: corev1.PodSpec{
			EphemeralContainers: ephemeralContainers,
		},
	}, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error creating ephemeral container: %v", err)
	}

	cmd.Log.Infof("Debugger %s created for container %s in pod %s", debuggerName, cmd.Target, pod.Name)

	// Wait for the ephemeral container to be ready
	err = cmd.waitForContainer(ctx, client, pod.Namespace, pod.Name, debuggerName)
	if err != nil {
		return fmt.Errorf("error waiting for debug container: %v", err)
	}

	// Execute the command in the container
	cmd.errorChan = make(chan error)
	return executeCommand(debuggerName, pod.Name, pod.Namespace, cmd.KubeConfig, cmd.errorChan, cmd.Log)
}

func (cmd *DebugCmd) waitForContainer(ctx context.Context, client kubernetes.Interface, namespace, podName, containerName string) error {
	return wait.PollImmediate(time.Second, time.Minute*2, func() (bool, error) {
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

func executeCommand(conatinerName, pod, namespace, kubeConfig string, errorChan chan error, log log.Logger) error {
	commandErrChan := make(chan error)
	execCmd := exec.Command("kubectl", "--kubeconfig", kubeConfig, "exec", "-n", namespace, pod, "-it", "-c", conatinerName, "--", "/bin/sh", "-c", "echo \"$DEBUG_MESSAGE\" && ash")
	// log.Infof("command: %s", execCmd.String())
	execCmd.Env = os.Environ()
	execCmd.Env = append(execCmd.Env)
	execCmd.Stdout = os.Stdout
	execCmd.Stdin = os.Stdin
	execCmd.Stderr = os.Stderr
	err := execCmd.Start()
	if err != nil {
		return err
	}
	if errorChan == nil {
		return execCmd.Wait()
	}

	go func() {
		commandErrChan <- execCmd.Wait()
	}()

	select {
	case err := <-errorChan:
		if execCmd.Process != nil {
			_ = execCmd.Process.Kill()
		}

		return fmt.Errorf("error port-forwarding: %w", err)
	case err := <-commandErrChan:
		if exitError, ok := lo.ErrorsAs[*exec.ExitError](err); ok {
			log.Errorf("Error executing command: %v", err)
			os.Exit(exitError.ExitCode())
		}

		return err
	}
}

func ephemeralContainerDebugMessage(version string) string {
	return "This is ephemeral container attached to your virtual cluster pod.\n\n" +
		"vCluster version: " + version + "\n\n" +
		"Please do not modify any files in /proc/1, as it may affect the virtual cluster runtime.\n" +
		"ETCD environment variables are already configured for you, so you may check your ETCD cluster state with etcdctl.\n\n" +
		"Your virtual cluster config is located in:\n" +
		"$ cat /proc/1/root/var/lib/vcluster/config.yaml\n\n" +
		"Useful ETCD debugging commands:\n" +
		"$ etcdctl member list\n" +
		"$ etcdctl endpoint health\n"
}
