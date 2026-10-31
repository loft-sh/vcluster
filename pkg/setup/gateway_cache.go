package setup

import (
	"strings"

	"github.com/loft-sh/vcluster/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

func gatewaySourceNamespaces(options *config.VirtualClusterConfig) map[string]cache.Config {
	if !options.Sync.FromHost.Gateways.Enabled {
		return nil
	}

	gatewayNamespaces := map[string]cache.Config{}
	for hostName := range options.Sync.FromHost.Gateways.Mappings.ByName {
		hostNamespace, _, hasName := strings.Cut(hostName, "/")
		if !hasName || hostNamespace == "" {
			continue
		}

		gatewayNamespaces[hostNamespace] = cache.Config{}
	}

	return gatewayNamespaces
}
