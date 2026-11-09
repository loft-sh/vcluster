package find

import (
	"fmt"
	"os"
	"path/filepath"

	vclusterconfig "github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
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

func createKubeClientConfigFromPath(kubeConfigPath string) clientcmd.ClientConfig {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(&clientcmd.ClientConfigLoadingRules{
		ExplicitPath: kubeConfigPath,
	}, &clientcmd.ConfigOverrides{})
}

func getStandaloneKubeClientConfig(vConfig *vclusterconfig.VirtualClusterConfig) (clientcmd.ClientConfig, error) {
	if vConfig == nil {
		return nil, fmt.Errorf("standalone config is nil")
	}

	kubeConfigPath := vConfig.Experimental.VirtualClusterKubeConfig.KubeConfig
	if kubeConfigPath == "" {
		dataDir := vConfig.ControlPlane.Standalone.DataDir
		if dataDir == "" {
			dataDir = constants.VClusterStandaloneDefaultDataDir
		}
		kubeConfigPath = filepath.Join(dataDir, "pki", "admin.conf")
	}

	if _, err := os.Stat(kubeConfigPath); err != nil {
		return nil, fmt.Errorf("stat standalone kubeconfig %s: %w", kubeConfigPath, err)
	}

	return createKubeClientConfigFromPath(kubeConfigPath), nil
}

func CreateKubeClient(context string) (kube.Interface, error) {
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
