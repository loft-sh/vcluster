// Suite: plugin-vcluster
// vCluster: legacy v1/v2 plugins (bootstrap-with-deployment, hooks, import-secrets).
// Lifecycle owned by this Describe's BeforeAll + DeferCleanup.
// Run:      just run-e2e 'plugin'
// NonDefault: plugin example images (bootstrap-with-deployment:v2) are amd64-only.
// These tests cannot run on macOS ARM (Kind on Apple Silicon) - CI only.
package e2e_next

import (
	"context"
	_ "embed"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/e2e-next/setup/lazyvcluster"
	"github.com/loft-sh/vcluster/e2e-next/test_integration/plugin"
	. "github.com/onsi/ginkgo/v2"
)

//go:embed vcluster-plugin.yaml
var pluginVClusterYAML string

const pluginVClusterName = "plugin-vcluster"

func init() { suitePluginVCluster() }

// Ordered: the outer Describe owns vCluster lifecycle via BeforeAll +
// DeferCleanup - Ginkgo only allows BeforeAll/AfterAll inside Ordered
// containers.
func suitePluginVCluster() {
	Describe("plugin-vcluster", labels.Integration, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx, pluginVClusterName, pluginVClusterYAML)
			})

			plugin.PluginSpec()
		},
	)
}
