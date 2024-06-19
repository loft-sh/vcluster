package localkubernetes

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
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
		c == ClusterTypeKIND ||
		c == ClusterTypeMinikube ||
		c == ClusterTypeK3D ||
		c == ClusterTypeOrbstack
}

func ExposeLocal(ctx context.Context, vClusterName, vClusterNamespace string, rawConfig *clientcmdapi.Config, vRawConfig *clientcmdapi.Config, service *corev1.Service, localPort int, log log.Logger) (string, error) {
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
	case ClusterTypeKIND:
		return kindProxy(ctx, vClusterName, vClusterNamespace, rawConfig, vRawConfig, service, localPort, timeout, log)
	case ClusterTypeMinikube:
		return minikubeProxy(ctx, vClusterName, vClusterNamespace, rawConfig, vRawConfig, service, localPort, timeout, log)
	case ClusterTypeK3D:
		return k3dProxy(ctx, vClusterName, vClusterNamespace, rawConfig, vRawConfig, service, localPort, timeout, log)
	default:
	}

	return "", nil
}

func CleanupLocal(vClusterName, vClusterNamespace string, rawConfig *clientcmdapi.Config, log log.Logger) error {
	clusterType := DetectClusterType(rawConfig)
	switch clusterType {
	case ClusterTypeMinikube:
		if containerExists(rawConfig.CurrentContext) {
			return cleanupProxy(vClusterName, vClusterNamespace, rawConfig, log)
		}

		return nil
	case ClusterTypeKIND:
		return cleanupProxy(vClusterName, vClusterNamespace, rawConfig, log)
	case ClusterTypeK3D:
		return cleanupProxy(vClusterName, vClusterNamespace, rawConfig, log)
	default:
	}

	return nil
}

func k3dProxy(ctx context.Context, vClusterName, vClusterNamespace string, rawConfig *clientcmdapi.Config, vRawConfig *clientcmdapi.Config, service *corev1.Service, localPort int, timeout time.Duration, log log.Logger) (string, error) {
	if len(service.Spec.Ports) == 0 {
		return "", fmt.Errorf("service has %d ports (expected 1 port)", len(service.Spec.Ports))
	}

	// see if we already have a proxy container running
	server, err := getServerFromExistingProxyContainer(ctx, vClusterName, vClusterNamespace, rawConfig, vRawConfig, service, log)
	if err != nil {
		return "", err
	} else if server != "" {
		return server, nil
	}

	k3dName := strings.TrimPrefix(rawConfig.CurrentContext, "k3d-")
	return createProxyContainer(ctx, vClusterName, vClusterNamespace, rawConfig, vRawConfig, service, localPort, timeout, "k3d-"+k3dName+"-server-0", "k3d-"+k3dName, log)
}

func minikubeProxy(ctx context.Context, vClusterName, vClusterNamespace string, rawConfig *clientcmdapi.Config, vRawConfig *clientcmdapi.Config, service *corev1.Service, localPort int, timeout time.Duration, log log.Logger) (string, error) {
	if len(service.Spec.Ports) == 0 {
		return "", fmt.Errorf("service has %d ports (expected 1 port)", len(service.Spec.Ports))
	}

	// check if docker driver or vm
	minikubeName := rawConfig.CurrentContext
	if containerExists(minikubeName) {
		// see if we already have a proxy container running
		server, err := getServerFromExistingProxyContainer(ctx, vClusterName, vClusterNamespace, rawConfig, vRawConfig, service, log)
		if err != nil {
			return "", err
		} else if server != "" {
			return server, nil
		}

		// create proxy container if missing
		return createProxyContainer(ctx, vClusterName, vClusterNamespace, rawConfig, vRawConfig, service, localPort, timeout, minikubeName, minikubeName, log)
	}

	// in case other type of driver (e.g. VM on linux) is used
	// check if the service is reacheable directly via the minikube IP
	c := rawConfig.Contexts[rawConfig.CurrentContext]
	if c != nil {
		s := rawConfig.Clusters[c.Cluster]
		if s != nil {
			u, err := url.Parse(s.Server)
			if err == nil {
				splitted := strings.Split(u.Host, ":")
				server := fmt.Sprintf("https://%s:%v", splitted[0], service.Spec.Ports[0].NodePort)

				// workaround for the fact that vcluster certificate is not made valid for the node IPs
				// but avoid modifying the passed config before the connection is tested
				testvConfig := vRawConfig.DeepCopy()
				for k := range testvConfig.Clusters {
					testvConfig.Clusters[k].CertificateAuthorityData = nil
					testvConfig.Clusters[k].InsecureSkipTLSVerify = true
				}

				// test local connection
				waitErr := wait.PollUntilContextTimeout(ctx, time.Second, timeout, true, func(ctx context.Context) (bool, error) {
					err = testConnectionWithServer(ctx, testvConfig, server)
					if err != nil {
						return false, nil
					}

					return true, nil
				})
				if waitErr != nil {
					return "", fmt.Errorf("test connection: %w %w", waitErr, err)
				}

				// now it's safe to modify the vRawConfig struct that was passed in as a pointer
				for k := range vRawConfig.Clusters {
					vRawConfig.Clusters[k].CertificateAuthorityData = nil
					vRawConfig.Clusters[k].InsecureSkipTLSVerify = true
				}

				return server, nil
			}
		}
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

func cleanupProxy(vClusterName, vClusterNamespace string, rawConfig *clientcmdapi.Config, log log.Logger) error {
	// construct proxy name
	proxyName := find.VClusterContextName(vClusterName, vClusterNamespace, rawConfig.CurrentContext)

	// check if proxy container already exists
	cmd := exec.Command(
		"docker",
		"stop",
		proxyName,
	)
	log.Infof("Stopping docker proxy...")
	_, _ = cmd.Output()
	return nil
}

func kindProxy(ctx context.Context, vClusterName, vClusterNamespace string, rawConfig *clientcmdapi.Config, vRawConfig *clientcmdapi.Config, service *corev1.Service, localPort int, timeout time.Duration, log log.Logger) (string, error) {
	if len(service.Spec.Ports) == 0 {
		return "", fmt.Errorf("service has %d ports (expected 1 port)", len(service.Spec.Ports))
	}

	// see if we already have a proxy container running
	server, err := getServerFromExistingProxyContainer(ctx, vClusterName, vClusterNamespace, rawConfig, vRawConfig, service, log)
	if err != nil {
		return "", err
	} else if server != "" {
		return server, nil
	}

	// name is prefixed with kind- and suffixed with -control-plane
	controlPlane := strings.TrimPrefix(rawConfig.CurrentContext, "kind-") + "-control-plane"
	return createProxyContainer(ctx, vClusterName, vClusterNamespace, rawConfig, vRawConfig, service, localPort, timeout, controlPlane, "kind", log)
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

func createProxyContainer(ctx context.Context, vClusterName, vClusterNamespace string, rawConfig *clientcmdapi.Config, vRawConfig *clientcmdapi.Config, service *corev1.Service, localPort int, timeout time.Duration, backendHost, network string, log log.Logger) (string, error) {
	// construct proxy name
	proxyName := find.VClusterContextName(vClusterName, vClusterNamespace, rawConfig.CurrentContext)

	// in general, we need to run this statement to expose the correct port for this
	// docker run -d -p LOCAL_PORT:NODE_PORT --rm -e "BACKEND_HOST=NAME-control-plane" -e "BACKEND_PORT=NODE_PORT" --network=NETWORK ghcr.io/loft-sh/docker-tcp-proxy
	cmd := exec.Command(
		"docker",
		"run",
		"-d",
		"-p",
		fmt.Sprintf("%v:%v", localPort, service.Spec.Ports[0].NodePort),
		"--rm",
		fmt.Sprintf("--name=%s", proxyName),
		"-e",
		fmt.Sprintf("BACKEND_HOST=%s", backendHost),
		"-e",
		fmt.Sprintf("BACKEND_PORT=%v", service.Spec.Ports[0].NodePort),
		fmt.Sprintf("--network=%s", network),
		"ghcr.io/loft-sh/docker-tcp-proxy",
	)
	log.Infof("Starting proxy container...")
	out, err := cmd.Output()
	if err != nil {
		return "", errors.Errorf("error starting kind proxy: %s %v", string(out), err)
	}

	server := fmt.Sprintf("https://127.0.0.1:%v", localPort)
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
	for k := range vRawConfig.Clusters {
		vRawConfig.Clusters[k].Server = server
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

func getServerFromExistingProxyContainer(ctx context.Context, vClusterName, vClusterNamespace string, rawConfig *clientcmdapi.Config, vRawConfig *clientcmdapi.Config, service *corev1.Service, log log.Logger) (string, error) {
	// construct proxy name
	proxyName := find.VClusterContextName(vClusterName, vClusterNamespace, rawConfig.CurrentContext)

	// check if proxy container already exists
	cmd := exec.Command(
		"docker",
		"inspect",
		proxyName,
		"-f",
		fmt.Sprintf("{{ index (index (index .HostConfig.PortBindings \"%v/tcp\") 0) \"HostPort\" }}", service.Spec.Ports[0].NodePort),
	)
	out, err := cmd.Output()
	if err == nil {
		localPort, err := strconv.Atoi(strings.TrimSpace(string(out)))
		if err == nil && localPort != 0 {
			server := fmt.Sprintf("https://127.0.0.1:%v", localPort)
			waitErr := wait.PollUntilContextTimeout(ctx, time.Second, time.Second*5, true, func(ctx context.Context) (bool, error) {
				err = testConnectionWithServer(ctx, vRawConfig, server)
				if err != nil {
					return false, nil
				}

				return true, nil
			})
			if waitErr != nil {
				// return err here as waitErr is only timed out
				return "", errors.Wrap(err, "test connection")
			}

			return server, nil
		}
	} else {
		log.Debugf("Error running docker inspect with go template: %v", err)
	}

	if containerExists(proxyName) {
		err := cleanupProxy(vClusterName, vClusterNamespace, rawConfig, log)
		if err != nil {
			return "", err
		}
	}

	return "", nil
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
