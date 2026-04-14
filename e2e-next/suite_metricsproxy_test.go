package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_integration/metricsproxy"
	. "github.com/onsi/ginkgo/v2"
)

func init() { suiteMetricsProxyVCluster() }

func suiteMetricsProxyVCluster() {
	Describe("metricsproxy-vcluster",
		cluster.Use(clusters.MetricsProxyVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			metricsproxy.MetricsProxySpec()
		},
	)
}
