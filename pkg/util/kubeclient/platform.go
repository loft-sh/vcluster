package kubeclient

import (
	"fmt"
	"net/url"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// PlatformRestConfigProvider is satisfied by platform.Client without requiring
// this package to import pkg/platform, which would create a circular dependency.
type PlatformRestConfigProvider interface {
	RestConfig(hostSuffix string) (*rest.Config, error)
}

// NewPlatformProxyClient creates a kubernetes client that talks to a vCluster
// through the platform's proxy REST endpoint.
func NewPlatformProxyClient(provider PlatformRestConfigProvider, project, name string, opts ...Option) (kubernetes.Interface, *rest.Config, error) {
	suffix := "/kubernetes/project/" + url.PathEscape(project) + "/virtualcluster/" + url.PathEscape(name)

	restConfig, err := provider.RestConfig(suffix)
	if err != nil {
		return nil, nil, fmt.Errorf("create platform proxy rest config: %w", err)
	}

	applyWrapTransport(restConfig, applyOptions(opts))

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("create platform proxy client: %w", err)
	}

	return client, restConfig, nil
}
