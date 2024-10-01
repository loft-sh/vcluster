package localkubernetes

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/util/kubeconfig"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func (c ClusterType) LocalKubernetes() bool {
	return c == ClusterTypeDockerDesktop ||
		c == ClusterTypeRancherDesktop ||
		c == ClusterTypeOrbstack
}

func ExposeLocal(ctx context.Context, rawConfig *clientcmdapi.Config, vRawConfig *clientcmdapi.Config, service *corev1.Service) (string, error) {
	// Timeout to wait for connection before falling back to port-forwarding
	timeout := time.Second * 30
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

func CreateBackgroundProxyContainer(ctx context.Context, vClusterName, vClusterNamespace string, rawConfig clientcmd.ClientConfig, vRawConfig *clientcmdapi.Config, localPort int, log log.Logger) (string, error) {
	rawConfigObj, err := rawConfig.RawConfig()
	if err != nil {
		return "", err
	}

	// write kube config to buffer
	physicalCluster, err := kubeconfig.ResolveKubeConfig(rawConfig)
	if err != nil {
		return "", fmt.Errorf("resolve kube config: %w", err)
	}

	// write a temporary kube file
	tempFile, err := os.CreateTemp("", "")
	if err != nil {
		return "", errors.Wrap(err, "create temp file")
	}
	_, err = tempFile.Write(physicalCluster)
	if err != nil {
		return "", errors.Wrap(err, "write kube config to temp file")
	}
	err = tempFile.Close()
	if err != nil {
		return "", errors.Wrap(err, "close temp file")
	}
	kubeConfigPath := tempFile.Name()

	// allow permissions for kube config path
	err = os.Chmod(kubeConfigPath, 0666)
	if err != nil {
		return "", fmt.Errorf("chmod temp file: %w", err)
	}

	// construct proxy name
	proxyName := find.VClusterConnectBackgroundProxyName(vClusterName, vClusterNamespace, rawConfigObj.CurrentContext)

	// check if the background proxy container for this vcluster is running and then remove it.
	_ = CleanupBackgroundProxy(proxyName, log)

	// build the command
	cmd := exec.Command(
		"docker",
		"run",
		"-d",
		"-v", fmt.Sprintf("%v:%v", kubeConfigPath, "/kube-config"),
		fmt.Sprintf("--name=%s", proxyName),
		"--network=host",
		"bitnami/kubectl:1.29",
		"port-forward",
		"svc/"+vClusterName,
		strconv.Itoa(localPort)+":443",
		"--kubeconfig", "/kube-config",
		"-n", vClusterNamespace,
	)
	log.Infof("Starting background proxy container...")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Errorf("error starting background proxy: %s %v", string(out), err)
	}
	server := fmt.Sprintf("https://127.0.0.1:%v", localPort)
	waitErr := wait.PollUntilContextTimeout(ctx, time.Second, time.Second*60, true, func(ctx context.Context) (bool, error) {
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
		return errors.Wrap(err, "retrieve default namespace")
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
