package localkubernetes

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/find"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func (c ClusterType) NodePortSupported() bool {
	return c == ClusterTypeDockerDesktop ||
		c == ClusterTypeRancherDesktop ||
		c == ClusterTypeKIND ||
		c == ClusterTypeMinikube ||
		c == ClusterTypeK3D
}

func ExposeLocal(vClusterName, vClusterNamespace string, rawConfig *clientcmdapi.Config, vRawConfig *clientcmdapi.Config, service *corev1.Service, localPort int, log log.Logger) (string, error) {
	clusterType := DetectClusterType(rawConfig)
	switch clusterType {
	case ClusterTypeDockerDesktop:
		return directConnection(vRawConfig, service)
	case ClusterTypeRancherDesktop:
		return directConnection(vRawConfig, service)
	case ClusterTypeKIND:
		return kindProxy(vClusterName, vClusterNamespace, rawConfig, vRawConfig, service, localPort, log)
	case ClusterTypeMinikube:
		return minikubeProxy(vClusterName, vClusterNamespace, rawConfig, vRawConfig, service, localPort, log)
	case ClusterTypeK3D:
		return k3dProxy(vClusterName, vClusterNamespace, rawConfig, vRawConfig, service, localPort, log)
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
	}

	return nil
}

func k3dProxy(vClusterName, vClusterNamespace string, rawConfig *clientcmdapi.Config, vRawConfig *clientcmdapi.Config, service *corev1.Service, localPort int, log log.Logger) (string, error) {
	if len(service.Spec.Ports) != 1 {
		return "", nil
	}

	// see if we already have a proxy container running
	server, err := getServerFromExistingProxyContainer(vClusterName, vClusterNamespace, rawConfig, vRawConfig, service, log)
	if err != nil {
		return "", err
	} else if server != "" {
		return server, nil
	}

	k3dName := strings.TrimPrefix(rawConfig.CurrentContext, "k3d-")
	return createProxyContainer(vClusterName, vClusterNamespace, rawConfig, vRawConfig, service, localPort, "k3d-"+k3dName+"-server-0", "k3d-"+k3dName, log)
}

func minikubeProxy(vClusterName, vClusterNamespace string, rawConfig *clientcmdapi.Config, vRawConfig *clientcmdapi.Config, service *corev1.Service, localPort int, log log.Logger) (string, error) {
	if len(service.Spec.Ports) != 1 {
		return "", nil
	}

	// check if docker driver or vm
	minikubeName := rawConfig.CurrentContext
	if containerExists(minikubeName) {
		// see if we already have a proxy container running
		server, err := getServerFromExistingProxyContainer(vClusterName, vClusterNamespace, rawConfig, vRawConfig, service, log)
		if err != nil {
			return "", err
		} else if server != "" {
			return server, nil
		}

		// create proxy container if missing
		return createProxyContainer(vClusterName, vClusterNamespace, rawConfig, vRawConfig, service, localPort, minikubeName, minikubeName, log)
	} else {
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

					err := testConnectionWithServer(testvConfig, server)
					if err == nil {
						// now it's safe to modify the vRawConfig struct that was passed in as a pointer
						for k := range vRawConfig.Clusters {
							vRawConfig.Clusters[k].CertificateAuthorityData = nil
							vRawConfig.Clusters[k].InsecureSkipTLSVerify = true
						}
						return server, nil
					}
				}
			}
		}

		return "", nil
	}
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

func kindProxy(vClusterName, vClusterNamespace string, rawConfig *clientcmdapi.Config, vRawConfig *clientcmdapi.Config, service *corev1.Service, localPort int, log log.Logger) (string, error) {
	if len(service.Spec.Ports) != 1 {
		return "", nil
	}

	// see if we already have a proxy container running
	server, err := getServerFromExistingProxyContainer(vClusterName, vClusterNamespace, rawConfig, vRawConfig, service, log)
	if err != nil {
		return "", err
	} else if server != "" {
		return server, nil
	}

	// name is prefixed with kind- and suffixed with -control-plane
	controlPlane := strings.TrimPrefix(rawConfig.CurrentContext, "kind-") + "-control-plane"
	return createProxyContainer(vClusterName, vClusterNamespace, rawConfig, vRawConfig, service, localPort, controlPlane, "kind", log)
}

func directConnection(vRawConfig *clientcmdapi.Config, service *corev1.Service) (string, error) {
	if len(service.Spec.Ports) != 1 {
		return "", nil
	}

	server := fmt.Sprintf("https://127.0.0.1:%v", service.Spec.Ports[0].NodePort)
	var err error
	waitErr := wait.PollImmediate(time.Second, time.Second*20, func() (bool, error) {
		err = testConnectionWithServer(vRawConfig, server)
		if err != nil {
			return false, nil
		}

		return true, nil
	})
	if waitErr != nil {
		return "", fmt.Errorf("test connection: %v %v", waitErr, err)
	}

	return server, nil
}

func createProxyContainer(vClusterName, vClusterNamespace string, rawConfig *clientcmdapi.Config, vRawConfig *clientcmdapi.Config, service *corev1.Service, localPort int, backendHost, network string, log log.Logger) (string, error) {
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
	waitErr := wait.PollImmediate(time.Second, time.Second*30, func() (bool, error) {
		err = testConnectionWithServer(vRawConfig, server)
		if err != nil {
			return false, nil
		}

		return true, nil
	})
	if waitErr != nil {
		return "", fmt.Errorf("test connection: %v %v", waitErr, err)
	}

	return server, nil
}

func testConnectionWithServer(vRawConfig *clientcmdapi.Config, server string) error {
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	_, err = kubeClient.CoreV1().Namespaces().Get(ctx, "default", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "retrieve default namespace")
	}

	return nil
}

func getServerFromExistingProxyContainer(vClusterName, vClusterNamespace string, rawConfig *clientcmdapi.Config, vRawConfig *clientcmdapi.Config, service *corev1.Service, log log.Logger) (string, error) {
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
			waitErr := wait.PollImmediate(time.Second, time.Second*5, func() (bool, error) {
				err = testConnectionWithServer(vRawConfig, server)
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

	// check if container exists
	found := containerExists(proxyName)
	if found {
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
		containerName,
	)
	_, err := cmd.Output()
	return err == nil
}
