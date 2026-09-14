package localkubernetes

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/util/kubeconfig"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const dockerInternalHostName = "host.docker.internal"

var ghcrDeniedErrorRe = regexp.MustCompile(`ghcr\.io.*\s*.*denied`)

func (c ClusterType) LocalKubernetes() bool {
	return c == ClusterTypeDockerDesktop ||
		c == ClusterTypeRancherDesktop ||
		c == ClusterTypeOrbstack
}

func ExposeLocal(ctx context.Context, rawConfig *clientcmdapi.Config, vRawConfig *clientcmdapi.Config, service *corev1.Service) (string, error) {
	// Timeout to wait for connection before falling back to port-forwarding
	timeout := time.Second * 5
	clusterType := DetectClusterType(rawConfig)
	switch clusterType {
	case ClusterTypeOrbstack:
		return directServiceConnection(ctx, vRawConfig, service, timeout)
	case ClusterTypeDockerDesktop:
		return directConnection(ctx, vRawConfig, service, timeout)
	case ClusterTypeRancherDesktop:
		return directConnection(ctx, vRawConfig, service, timeout)
	default:
	}

	return "", nil
}

func CleanupBackgroundProxy(proxyName string, log log.Logger) error {
	// check if background proxy container already exists
	if containerExists(proxyName) {
		// remove background proxy container
		cmd := exec.Command(
			"docker",
			"container",
			"rm",
			proxyName,
			"-f",
		)
		log.Infof("Stopping background proxy...")
		_, _ = cmd.Output()
	}
	return nil
}

func directServiceConnection(ctx context.Context, vRawConfig *clientcmdapi.Config, service *corev1.Service, timeout time.Duration) (string, error) {
	if len(service.Spec.Ports) == 0 {
		return "", fmt.Errorf("service has %d ports (expected 1 port)", len(service.Spec.Ports))
	}

	server := fmt.Sprintf("https://%s:443", service.Spec.ClusterIP)
	var err error
	waitErr := wait.PollUntilContextTimeout(ctx, time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		err = testConnectionWithServer(ctx, vRawConfig, server)
		if err != nil {
			return false, nil
		}

		return true, nil
	})
	if waitErr != nil {
		return "", fmt.Errorf("test connection: %w %w", waitErr, err)
	}

	return server, nil
}

func directConnection(ctx context.Context, vRawConfig *clientcmdapi.Config, service *corev1.Service, timeout time.Duration) (string, error) {
	if service.Spec.Type != corev1.ServiceTypeNodePort {
		return "", nil
	}
	if len(service.Spec.Ports) == 0 {
		return "", fmt.Errorf("service has %d ports (expected 1 port)", len(service.Spec.Ports))
	}

	server := fmt.Sprintf("https://127.0.0.1:%v", service.Spec.Ports[0].NodePort)
	var err error
	waitErr := wait.PollUntilContextTimeout(ctx, time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		err = testConnectionWithServer(ctx, vRawConfig, server)
		if err != nil {
			return false, nil
		}

		return true, nil
	})
	if waitErr != nil {
		return "", fmt.Errorf("test connection: %w %w", waitErr, err)
	}

	return server, nil
}

// CreateBackgroundProxyContainer runs kubectl port-forward in a docker container, forwarding from the vcluster service
// on the host cluster to a port matching the kubernetes context for the virtual cluster.
func CreateBackgroundProxyContainer(_ context.Context, vClusterName, vClusterNamespace string, proxyImage string, rawConfig clientcmd.ClientConfig, localPort int, log log.Logger) (string, error) {
	rawConfigObj, err := rawConfig.RawConfig()
	if err != nil {
		return "", err
	}

	physicalRawConfig, err := kubeconfig.ResolveKubeConfig(rawConfig)
	if err != nil {
		return "", fmt.Errorf("resolve kube config: %w", err)
	}

	// construct proxy name
	proxyName := find.VClusterConnectBackgroundProxyName(vClusterName, vClusterNamespace, rawConfigObj.CurrentContext)

	// check if the background proxy container for this vcluster is running and then remove it.
	_ = CleanupBackgroundProxy(proxyName, log)

	cmd, err := buildDockerCommand(physicalRawConfig, proxyName, vClusterName, vClusterNamespace, proxyImage, localPort)
	if err != nil {
		return "", fmt.Errorf("build docker command: %w", err)
	}

	log.Infof("Starting background proxy container...")
	if out, err := cmd.CombinedOutput(); err != nil {
		output := string(out)
		if ghcrDeniedErrorRe.MatchString(output) {
			return "", fmt.Errorf("unabled to find image '%s' locally and pulling the image was denied. If you are logged into ghcr.io, ensure that your credentials are valid or logout by running 'docker logout ghcr.io'", proxyImage)
		}
		return "", fmt.Errorf("error starting background proxy: %s %w", output, err)
	}

	return fmt.Sprintf("https://127.0.0.1:%v", localPort), nil
}

// build a different docker command for darwin vs. everything else
func buildDockerCommand(physicalRawConfig clientcmdapi.Config, proxyName, vClusterName, vClusterNamespace string, proxyImage string, localPort int) (*exec.Cmd, error) {
	// write a temporary kube file
	tempFile, err := os.CreateTemp("", "")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	kubeConfigPath := tempFile.Name()

	dockerArgs := []string{
		"run",
		"--rm",
		"-d",
		"-v", fmt.Sprintf("%v:%v", kubeConfigPath, "/kube-config"),
		fmt.Sprintf("--name=%s", proxyName),
		"--entrypoint=/vcluster",
	}

	// For non-linux, update the kube config to point to the special host.docker.internal and don't use
	// host networking.
	if runtime.GOOS != "linux" {
		physicalRawConfig, err = updateConfigForDockerToHost(physicalRawConfig)
		if err != nil {
			return nil, fmt.Errorf("update config: %w", err)
		}

		dockerArgs = append(dockerArgs,
			"-p",
			fmt.Sprintf("%d:8443", localPort),
			proxyImage,
			"port-forward",
			"svc/"+vClusterName,
			"--address=0.0.0.0",
			"8443:443",
			"--kubeconfig", "/kube-config",
			"-n", vClusterNamespace,
		)
	} else {
		dockerArgs = append(dockerArgs,
			"--network=host",
			proxyImage,
			"port-forward",
			"svc/"+vClusterName,
			"--address=0.0.0.0",
			strconv.Itoa(localPort)+":443",
			"--kubeconfig", "/kube-config",
			"-n", vClusterNamespace,
		)
	}

	cmd := exec.Command("docker", dockerArgs...)

	// write kube config to buffer
	physicalCluster, err := clientcmd.Write(physicalRawConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	if _, err = tempFile.Write(physicalCluster); err != nil {
		return nil, fmt.Errorf("write kube config to temp file: %w", err)
	}

	if err = tempFile.Close(); err != nil {
		return nil, fmt.Errorf("close temp file: %w", err)
	}

	// allow permissions for kube config path
	if err = os.Chmod(kubeConfigPath, 0666); err != nil {
		return nil, fmt.Errorf("chmod temp file: %w", err)
	}

	return cmd, nil
}

// Update the configuration for the local cluster to be able to reach the host via the special host.docker.internal address
func updateConfigForDockerToHost(rawConfig clientcmdapi.Config) (clientcmdapi.Config, error) {
	updated := rawConfig.DeepCopy()

	if updated.Clusters == nil {
		return clientcmdapi.Config{}, fmt.Errorf("config missing clusters")
	}

	if _, ok := updated.Clusters["local"]; !ok {
		return clientcmdapi.Config{}, fmt.Errorf("config missing local cluster")
	}

	localCluster := updated.Clusters["local"]
	localCluster.InsecureSkipTLSVerify = true
	localCluster.CertificateAuthorityData = nil

	localCluster.Server = strings.ReplaceAll(localCluster.Server, "127.0.0.1", dockerInternalHostName)
	localCluster.Server = strings.ReplaceAll(localCluster.Server, "0.0.0.0", dockerInternalHostName)
	localCluster.Server = strings.ReplaceAll(localCluster.Server, "localhost", dockerInternalHostName)

	return *updated, nil
}

func IsDockerInstalledAndUpAndRunning() bool {
	cmd := exec.Command(
		"docker",
		"ps",
	)
	_, err := cmd.Output()
	return err == nil
}

func testConnectionWithServer(ctx context.Context, vRawConfig *clientcmdapi.Config, server string) error {
	vRawConfig = vRawConfig.DeepCopy()
	for _, cluster := range vRawConfig.Clusters {
		if cluster == nil {
			continue
		}

		cluster.Server = server
	}

	restConfig, err := clientcmd.NewDefaultClientConfig(*vRawConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	_, err = kubeClient.CoreV1().Namespaces().Get(ctx, "default", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("retrieve default namespace: %w", err)
	}

	return nil
}

func containerExists(containerName string) bool {
	cmd := exec.Command(
		"docker",
		"inspect",
		"--type=container",
		containerName,
	)
	_, err := cmd.Output()
	return err == nil
}
