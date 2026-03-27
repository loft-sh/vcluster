package kubeclient

import (
	"fmt"
	"net/http"

	"github.com/loft-sh/vcluster/pkg/platform/kube"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// HostClientConfig returns a ClientConfig for the given kubeconfig context using
// the default loading rules. An empty contextName uses the current context.
func HostClientConfig(contextName string) clientcmd.ClientConfig {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{CurrentContext: contextName},
	)
}

// NewHostClusterClient creates a kube client for the given host cluster context.
// An empty contextName uses the current context from the local kubeconfig.
func NewHostClusterClient(contextName string, opts ...Option) (kube.Interface, clientcmd.ClientConfig, error) {
	cfg := HostClientConfig(contextName)

	restConfig, err := cfg.ClientConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("get host cluster client config: %w", err)
	}

	applyWrapTransport(restConfig, applyOptions(opts))

	kubeClient, err := kube.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("create host cluster client: %w", err)
	}

	return kubeClient, cfg, nil
}

// NewClientsetForContext creates a *kubernetes.Clientset for the given kubeconfig context.
// Pass an empty contextName to use the active context from the default kubeconfig.
func NewClientsetForContext(contextName string, opts ...Option) (*kubernetes.Clientset, error) {
	cfg := HostClientConfig(contextName)

	restConfig, err := cfg.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("there is an error loading your current kube config (%w), please make sure you have access to a kubernetes cluster and the command `kubectl get namespaces` is working", err)
	}

	applyWrapTransport(restConfig, applyOptions(opts))

	return kubernetes.NewForConfig(restConfig)
}

// applyWrapTransport chains o.wrapTransport onto restConfig.WrapTransport if set.
func applyWrapTransport(restConfig *rest.Config, o *clientOptions) {
	if o.wrapTransport == nil {
		return
	}
	wrapFn := o.wrapTransport
	prior := restConfig.WrapTransport
	restConfig.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if prior != nil {
			rt = prior(rt)
		}
		return wrapFn(rt)
	}
}
