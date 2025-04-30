package find

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/platform/kube"
	"k8s.io/client-go/tools/clientcmd"
)

func createKubeClientConfig(context string) clientcmd.ClientConfig {
	configOverrides := &clientcmd.ConfigOverrides{
		CurrentContext: context,
	}
	kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), configOverrides)
	return kubeClientConfig
}

func createKubeClient(context string) (kube.Interface, error) {
	kubeClientConfig := createKubeClientConfig(context)
	restConfig, err := kubeClientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kube client config: %w", err)
	}
	kubeClient, err := kube.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kube client: %w", err)
	}

	return kubeClient, nil
}
