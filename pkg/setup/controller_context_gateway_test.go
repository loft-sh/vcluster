package setup

import (
	"testing"

	rootconfig "github.com/loft-sh/vcluster/config"
	pkgconfig "github.com/loft-sh/vcluster/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestLocalCacheWatchesFromHostGatewayMappingSourceNamespaces(t *testing.T) {
	options := &pkgconfig.VirtualClusterConfig{}
	options.HostNamespace = "loft-default-v-test"
	options.Sync.FromHost.Gateways.Enabled = true
	options.Sync.FromHost.Gateways.Mappings.ByName = map[string]string{
		"platform-gateways/*":          "shared-gateways/*",
		"platform-ingress/shared-edge": "shared-gateways/shared-edge",
	}

	cacheOptions := getLocalCacheOptions(options)
	if _, ok := cacheOptions.DefaultNamespaces["loft-default-v-test"]; !ok {
		t.Fatalf("expected default local cache to watch host namespace, got %#v", cacheOptions.DefaultNamespaces)
	}
	for _, namespace := range []string{"platform-gateways", "platform-ingress"} {
		if _, ok := cacheOptions.DefaultNamespaces[namespace]; ok {
			t.Fatalf("expected default local cache not to watch gateway source namespace %q, got %#v", namespace, cacheOptions.DefaultNamespaces)
		}
	}

	gatewayNamespaces := gatewayCacheNamespaces(t, cacheOptions)
	for _, namespace := range []string{"loft-default-v-test", "platform-gateways", "platform-ingress"} {
		if _, ok := gatewayNamespaces[namespace]; !ok {
			t.Fatalf("expected Gateway cache to watch namespace %q, got %#v", namespace, gatewayNamespaces)
		}
	}
}

func TestLocalCacheDoesNotAddVirtualGatewayMappingNamespaces(t *testing.T) {
	options := &pkgconfig.VirtualClusterConfig{}
	options.HostNamespace = "loft-default-v-test"
	options.Sync.FromHost.Gateways.Enabled = true
	options.Sync.FromHost.Gateways.Mappings = rootconfig.FromHostMappings{ByName: map[string]string{
		"platform-gateways/public-web": "shared-gateways/shared-web",
	}}

	cacheOptions := getLocalCacheOptions(options)
	if _, ok := cacheOptions.DefaultNamespaces["shared-gateways"]; ok {
		t.Fatalf("expected default local cache not to watch virtual gateway mapping namespace, got %#v", cacheOptions.DefaultNamespaces)
	}
	if _, ok := gatewayCacheNamespaces(t, cacheOptions)["shared-gateways"]; ok {
		t.Fatalf("expected Gateway cache to watch host source namespaces only, got %#v", cacheOptions.ByObject)
	}
}

func gatewayCacheNamespaces(t *testing.T, cacheOptions cache.Options) map[string]cache.Config {
	t.Helper()

	for obj, byObject := range cacheOptions.ByObject {
		if _, ok := obj.(*gatewayv1.Gateway); ok {
			return byObject.Namespaces
		}
	}

	t.Fatalf("expected Gateway cache ByObject config, got %#v", cacheOptions.ByObject)
	return nil
}
