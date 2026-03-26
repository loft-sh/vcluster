// Suite: e2e_metrics_proxy
// Matches: test/e2e_metrics_proxy/
// vCluster: MetricsProxyVCluster (integrations.metricsServer.enabled: true)
// PreSetup: installs metrics-server on host cluster via Helm
// Run:      just run-e2e '/metricsproxy-vcluster/'
package e2e_next

import (
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/metricsproxy"
)

var (
	_ = metricsproxy.DescribeMetricsProxy(clusters.MetricsProxyVCluster)
)
