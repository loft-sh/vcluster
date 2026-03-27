package kubeclient

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// NewVClusterClient creates a kubernetes client using cfg.
//
// When contextName is non-empty, the client is configured for that specific
// context within cfg (e.g. a vCluster context embedded in the host kubeconfig).
// When contextName is empty, cfg's current context is used as-is — useful when
// the caller has already constructed a ClientConfig pointing at the right target.
func NewVClusterClient(cfg clientcmd.ClientConfig, contextName string, opts ...Option) (kubernetes.Interface, error) {
	var clientCfg clientcmd.ClientConfig
	if contextName != "" {
		rawConfig, err := cfg.RawConfig()
		if err != nil {
			return nil, fmt.Errorf("load kubeconfig: %w", err)
		}
		if _, ok := rawConfig.Contexts[contextName]; !ok {
			return nil, fmt.Errorf("context %q not found in kubeconfig", contextName)
		}
		clientCfg = clientcmd.NewDefaultClientConfig(rawConfig, &clientcmd.ConfigOverrides{
			CurrentContext: contextName,
		})
	} else {
		clientCfg = cfg
	}

	restConfig, err := clientCfg.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("create vcluster rest config: %w", err)
	}

	applyWrapTransport(restConfig, applyOptions(opts))

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create vcluster client: %w", err)
	}

	return client, nil
}

// NewVClusterClientFromConfig creates a kubernetes client directly from a raw
// clientcmdapi.Config. This is useful when the caller has already assembled the
// full config (e.g. from a Secret) and wants to target its current context.
func NewVClusterClientFromConfig(rawConfig clientcmdapi.Config, opts ...Option) (kubernetes.Interface, error) {
	cfg := clientcmd.NewDefaultClientConfig(rawConfig, &clientcmd.ConfigOverrides{})
	return NewVClusterClient(cfg, "", opts...)
}
