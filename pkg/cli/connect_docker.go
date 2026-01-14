package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// dockerContainerDetails represents relevant info from docker inspect
type dockerContainerDetails struct {
	ID              string                 `json:"Id,omitempty"`
	State           dockerContainerState   `json:"State,omitempty"`
	NetworkSettings dockerContainerNetwork `json:"NetworkSettings,omitempty"`
	Config          dockerContainerConfig  `json:"Config,omitempty"`
}

type dockerContainerState struct {
	Status    string `json:"Status,omitempty"`
	Running   bool   `json:"Running,omitempty"`
	StartedAt string `json:"StartedAt,omitempty"`
}

type dockerContainerNetwork struct {
	Ports map[string][]dockerContainerPort `json:"Ports,omitempty"`
}

type dockerContainerPort struct {
	HostIP   string `json:"HostIp,omitempty"`
	HostPort string `json:"HostPort,omitempty"`
}

type dockerContainerConfig struct {
	Env []string `json:"Env,omitempty"`
}

type connectDocker struct {
	*flags.GlobalFlags
	*ConnectOptions

	log log.Logger
}

func ConnectDocker(ctx context.Context, options *ConnectOptions, globalFlags *flags.GlobalFlags, vClusterName string, command []string, log log.Logger) error {
	cmd := &connectDocker{
		GlobalFlags:    globalFlags,
		ConnectOptions: options,
		log:            log,
	}

	return cmd.connect(ctx, vClusterName, command)
}

func (cmd *connectDocker) connect(ctx context.Context, vClusterName string, command []string) error {
	containerName := getControlPlaneContainerName(vClusterName)

	// find the docker container
	cmd.log.Infof("Finding docker container %s...", containerName)
	containerDetails, err := cmd.inspectDockerContainer(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to find vcluster container %s: %w", containerName, err)
	}

	// check if container is running
	if !containerDetails.State.Running {
		return fmt.Errorf("vcluster container %s is not running (status: %s)", containerName, containerDetails.State.Status)
	}

	// get the exposed port for 8443
	hostPort, err := cmd.getExposedPort(containerDetails, "8443/tcp")
	if err != nil {
		return fmt.Errorf("failed to get exposed port: %w", err)
	}
	cmd.log.Debugf("Found exposed port %s for vcluster container %s", hostPort, containerName)

	// get the kubeconfig from the container
	kubeConfig, err := getDockerVClusterKubeConfig(ctx, vClusterName, hostPort, cmd.log)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	// set context name
	if cmd.KubeConfigContextName == "" {
		cmd.KubeConfigContextName = "vcluster-docker_" + vClusterName
	}

	// exchange context name
	err = cmd.exchangeContextName(kubeConfig)
	if err != nil {
		return err
	}

	// wait for vCluster to become ready (unless just printing)
	if !cmd.ConnectOptions.Print {
		err = cmd.waitForVCluster(ctx, vClusterName, *kubeConfig)
		if err != nil {
			return fmt.Errorf("failed connecting to vcluster: %w", err)
		}
	}

	// check if we should execute command
	if len(command) > 0 {
		return executeCommand(*kubeConfig, command, nil, cmd.log)
	}

	// write kube config
	return writeKubeConfig(kubeConfig, vClusterName, cmd.ConnectOptions, cmd.GlobalFlags, false, cmd.log)
}

func (cmd *connectDocker) inspectDockerContainer(ctx context.Context, containerName string) (*dockerContainerDetails, error) {
	args := []string{"inspect", "--type", "container", containerName}
	out, err := exec.CommandContext(ctx, "docker", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("docker inspect failed: %w", err)
	}

	var containerDetails []dockerContainerDetails
	err = json.Unmarshal(out, &containerDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to parse docker inspect output: %w", err)
	}
	if len(containerDetails) == 0 {
		return nil, fmt.Errorf("container %s not found", containerName)
	}

	return &containerDetails[0], nil
}

func (cmd *connectDocker) getExposedPort(containerDetails *dockerContainerDetails, containerPort string) (string, error) {
	ports, ok := containerDetails.NetworkSettings.Ports[containerPort]
	if !ok || len(ports) == 0 {
		return "", fmt.Errorf("port %s is not exposed", containerPort)
	}

	hostPort := ports[0].HostPort
	if hostPort == "" {
		return "", fmt.Errorf("port %s has no host port mapping", containerPort)
	}

	return hostPort, nil
}

func getDockerVClusterKubeConfig(ctx context.Context, vClusterName string, hostPort string, log log.Logger) (*clientcmdapi.Config, error) {
	// The kubeconfig in standalone mode is written to /data/kubeconfig.yaml
	// We retrieve it from the container
	args := []string{"exec", getControlPlaneContainerName(vClusterName), "cat", "/var/lib/vcluster/kubeconfig.yaml"}

	var kubeConfigBytes []byte
	var err error

	// Poll until the kubeconfig is available (vcluster might still be starting up)
	log.Infof("Waiting for vCluster kubeconfig to be available...")
	start := time.Now()
	waitErr := wait.PollUntilContextTimeout(ctx, time.Second*2, time.Minute*5, true, func(ctx context.Context) (bool, error) {
		// after 10 seconds, check if the vCluster failed
		if time.Since(start) > time.Second*10 && isVClusterFailed(ctx, vClusterName) {
			return false, fmt.Errorf("vCluster failed: %s. \nvCluster failed to start, please check the logs above for more information", getVClusterLogs(ctx, vClusterName))
		}

		kubeConfigBytes, err = exec.CommandContext(ctx, "docker", args...).Output()
		if err != nil {
			log.Debugf("Kubeconfig not yet available: %v", err)
			return false, nil
		}
		return true, nil
	})
	if waitErr != nil {
		return nil, fmt.Errorf("timeout waiting for kubeconfig: %w (last error: %w)", waitErr, err)
	}

	// parse the kubeconfig
	kubeConfig, err := clientcmd.Load(kubeConfigBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	// update the server address to point to localhost:hostPort
	for _, cluster := range kubeConfig.Clusters {
		if cluster != nil {
			cluster.Server = "https://localhost:" + hostPort
			cluster.CertificateAuthorityData = nil
			cluster.InsecureSkipTLSVerify = true
		}
	}

	return kubeConfig, nil
}

// exchangeContextName updates the kubeconfig to use the configured context name
func (cmd *connectDocker) exchangeContextName(kubeConfig *clientcmdapi.Config) error {
	if kubeConfig == nil {
		return fmt.Errorf("nil kubeConfig")
	}
	if kubeConfig.Clusters == nil || kubeConfig.Contexts == nil || kubeConfig.AuthInfos == nil {
		return fmt.Errorf("kubeconfig is missing required fields")
	}

	// update cluster name
	for k, cluster := range kubeConfig.Clusters {
		kubeConfig.Clusters[cmd.KubeConfigContextName] = cluster
		if k != cmd.KubeConfigContextName {
			delete(kubeConfig.Clusters, k)
		}
		break
	}

	// update context
	for k, ctx := range kubeConfig.Contexts {
		if ctx == nil {
			continue
		}
		ctx.Cluster = cmd.KubeConfigContextName
		ctx.AuthInfo = cmd.KubeConfigContextName
		kubeConfig.Contexts[cmd.KubeConfigContextName] = ctx
		if k != cmd.KubeConfigContextName {
			delete(kubeConfig.Contexts, k)
		}
		break
	}

	// update authinfo
	for k, authInfo := range kubeConfig.AuthInfos {
		if authInfo == nil {
			continue
		}
		kubeConfig.AuthInfos[cmd.KubeConfigContextName] = authInfo
		if k != cmd.KubeConfigContextName {
			delete(kubeConfig.AuthInfos, k)
		}
		break
	}

	kubeConfig.CurrentContext = cmd.KubeConfigContextName
	return nil
}

func (cmd *connectDocker) waitForVCluster(ctx context.Context, vClusterName string, kubeConfig clientcmdapi.Config) error {
	cmd.log.Infof("Waiting for vCluster to become ready...")

	restConfig, err := clientcmd.NewDefaultClientConfig(kubeConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return fmt.Errorf("failed to create rest config: %w", err)
	}

	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create kube client: %w", err)
	}

	err = wait.PollUntilContextTimeout(ctx, time.Second, time.Minute*3, true, func(ctx context.Context) (bool, error) {
		ctx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()

		// check if we can reach the API server by getting the default service account
		_, err := kubeClient.CoreV1().ServiceAccounts("default").Get(ctx, "default", metav1.GetOptions{})
		if err != nil {
			cmd.log.Debugf("vCluster not ready yet: %v", err)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("timeout waiting for vcluster to become ready: %w", err)
	}

	cmd.log.Donef("vCluster is ready")
	return nil
}

func getVClusterLogs(ctx context.Context, vClusterName string) string {
	args := []string{"exec", getControlPlaneContainerName(vClusterName), "journalctl", "-u", "vcluster.service", "--no-pager", "-e"}
	out, _ := exec.CommandContext(ctx, "docker", args...).Output()
	return string(out)
}

func isVClusterFailed(ctx context.Context, vClusterName string) bool {
	args := []string{"exec", getControlPlaneContainerName(vClusterName), "systemctl", "show", "vcluster.service", "--property=MainPID", "--value"}
	out, _ := exec.CommandContext(ctx, "docker", args...).Output()
	return strings.TrimSpace(string(out)) == "0"
}
