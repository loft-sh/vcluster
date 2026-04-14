package clusters

import (
	_ "embed"

	setup "github.com/loft-sh/vcluster/e2e-next/setup"
)

// MetricsProxyVCluster has integrations.metricsServer.enabled to test the
// metrics proxy integration. PreSetup installs metrics-server on the host
// cluster before the vCluster is provisioned.

//go:embed vcluster-metricsproxy.yaml
var metricsProxyVClusterYAML string

var (
	MetricsProxyVClusterName = "metricsproxy-vcluster"
	MetricsProxyVCluster     = registerWith(MetricsProxyVClusterName, metricsProxyVClusterYAML,
		[]RegisterOption{WithPreSetup(setup.MetricsServerPreSetup())},
	)
)
