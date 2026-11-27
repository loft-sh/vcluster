package setup

import (
	"testing"

	rootconfig "github.com/loft-sh/vcluster/config"
	pkgconfig "github.com/loft-sh/vcluster/pkg/config"
)

func TestLocalCacheDoesNotWatchKubeconfigExportDestinations(t *testing.T) {
	options := &pkgconfig.VirtualClusterConfig{}
	options.HostNamespace = "tenant-host"
	options.ExportKubeConfig.AdditionalSecrets = []rootconfig.ExportKubeConfigAdditionalSecretReference{
		{Name: "tenant-export", Namespace: "central"},
		{Name: "other-export", Namespace: "another-central"},
	}

	cacheOptions := getLocalCacheOptions(options)
	if len(cacheOptions.DefaultNamespaces) != 1 {
		t.Fatalf("export expanded syncer watches: %#v", cacheOptions.DefaultNamespaces)
	}
	if _, ok := cacheOptions.DefaultNamespaces["tenant-host"]; !ok {
		t.Fatal("host namespace is not watched")
	}
}
