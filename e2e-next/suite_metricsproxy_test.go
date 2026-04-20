// Suite: metricsproxy-vcluster
// vCluster: integrations.metricsServer.enabled. PreSetup installs
// metrics-server on host cluster via Helm. Lifecycle owned by this
// Describe's BeforeAll + DeferCleanup.
// Run:      just run-e2e 'metricsproxy'
package e2e_next

import (
	"context"
	_ "embed"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/setup"
	"github.com/loft-sh/vcluster/e2e-next/setup/lazyvcluster"
	"github.com/loft-sh/vcluster/e2e-next/test_integration/metricsproxy"
	. "github.com/onsi/ginkgo/v2"
)

//go:embed vcluster-metricsproxy.yaml
var metricsProxyVClusterYAML string

const metricsProxyVClusterName = "metricsproxy-vcluster"

func init() { suiteMetricsProxyVCluster() }

// Ordered: the outer Describe owns vCluster lifecycle via BeforeAll +
// DeferCleanup - Ginkgo only allows BeforeAll/AfterAll inside Ordered
// containers.
func suiteMetricsProxyVCluster() {
	Describe("metricsproxy-vcluster", Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx,
					metricsProxyVClusterName,
					metricsProxyVClusterYAML,
					lazyvcluster.WithPreSetup(setup.MetricsServerPreSetup()),
				)
			})

			metricsproxy.MetricsProxySpec()
		},
	)
}
