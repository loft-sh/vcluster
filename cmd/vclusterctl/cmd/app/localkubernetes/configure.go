package localkubernetes

import (
	"context"
	"fmt"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/find"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func ExposeLocal(vClusterName, vClusterNamespace string, rawConfig *clientcmdapi.Config, vRawConfig *clientcmdapi.Config, service *corev1.Service, localPort int, log log.Logger) (string, error) {
	clusterType := DetectClusterType(rawConfig)
	switch clusterType {
	case ClusterTypeDockerDesktop:
		return directExposure(vRawConfig, service)
	case ClusterTypeRancherDesktop:
		return directExposure(vRawConfig, service)
	case ClusterTypeKIND:
		return kindProxy(vClusterName, vClusterNamespace, rawConfig, vRawConfig, service, localPort, log)
	}

	return "", nil
}

func CleanupLocal(vClusterName, vClusterNamespace string, rawConfig *clientcmdapi.Config, log log.Logger) error {
	clusterType := DetectClusterType(rawConfig)
	switch clusterType {
	case ClusterTypeKIND:
		return cleanupKindProxy(vClusterName, vClusterNamespace, rawConfig, log)
	}

	return nil
}

func cleanupKindProxy(vClusterName, vClusterNamespace string, rawConfig *clientcmdapi.Config, log log.Logger) error {
	// construct proxy name
	proxyName := find.VClusterContextName(vClusterName, vClusterNamespace, rawConfig.CurrentContext)

	// check if proxy container already exists
	cmd := exec.Command(
		"docker",
		"stop",
		proxyName,
	)
	log.Infof("Stopping kind proxy...")
	_, _ = cmd.Output()
	return nil
}

func kindProxy(vClusterName, vClusterNamespace string, rawConfig *clientcmdapi.Config, vRawConfig *clientcmdapi.Config, service *corev1.Service, localPort int, log log.Logger) (string, error) {
	if len(service.Spec.Ports) != 1 {
		return "", nil
	}

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
		localPort, err = strconv.Atoi(strings.TrimSpace(string(out)))
		if err == nil && localPort != 0 {
			server := fmt.Sprintf("https://localhost:%v", localPort)
			err = testConnectionWithServer(vRawConfig, server)
			if err != nil {
				return "", errors.Wrap(err, "test connection")
			}

			return server, nil
		}
	} else {
		log.Debugf("Error running docker inspect: %v", err)

		// in general, we need to run this statement to expose the correct port for this
		// docker run -d -p LOCAL_PORT:NODE_PORT --rm -e "BACKEND_HOST=NAME-control-plane" -e "BACKEND_PORT=NODE_PORT" --network=kind ghcr.io/loft-sh/docker-tcp-proxy
		controlPlane := strings.TrimPrefix(rawConfig.CurrentContext, "kind-")
		cmd = exec.Command(
			"docker",
			"run",
			"-d",
			"-p",
			fmt.Sprintf("%v:%v", localPort, service.Spec.Ports[0].NodePort),
			"--rm",
			fmt.Sprintf("--name=%s", proxyName),
			"-e",
			fmt.Sprintf("BACKEND_HOST=%s-control-plane", controlPlane),
			"-e",
			fmt.Sprintf("BACKEND_PORT=%v", service.Spec.Ports[0].NodePort),
			"--network=kind",
			"ghcr.io/loft-sh/docker-tcp-proxy",
		)
		log.Infof("Starting proxy container...")
		out, err = cmd.Output()
		if err != nil {
			return "", errors.Errorf("error starting kind proxy: %s %v", string(out), err)
		}
	}

	server := fmt.Sprintf("https://localhost:%v", localPort)
	waitErr := wait.PollImmediate(time.Second, time.Second*20, func() (done bool, err error) {
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

func directExposure(vRawConfig *clientcmdapi.Config, service *corev1.Service) (string, error) {
	if len(service.Spec.Ports) != 1 {
		return "", nil
	}

	server := fmt.Sprintf("https://localhost:%v", service.Spec.Ports[0].NodePort)
	err := testConnectionWithServer(vRawConfig, server)
	if err != nil {
		return "", err
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	_, err = kubeClient.CoreV1().Namespaces().Get(ctx, "default", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "retrieve default namespace")
	}

	return nil
}
