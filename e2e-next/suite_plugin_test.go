package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/e2e-next/test_integration/plugin"
	. "github.com/onsi/ginkgo/v2"
)

func init() { suitePluginVCluster() }

func suitePluginVCluster() {
	Describe("plugin-vcluster", labels.Integration,
		cluster.Use(clusters.PluginVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			plugin.PluginSpec()
		},
	)
}
