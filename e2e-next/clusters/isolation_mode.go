package clusters

import _ "embed"

// IsolationModeVCluster enables podSecurityStandard (baseline), resourceQuota,
// limitRange, and networkPolicy for isolation mode tests.

//go:embed vcluster-isolation-mode.yaml
var isolationModeVClusterYAML string

var (
	IsolationModeVClusterName = "isolation-mode-vcluster"
	IsolationModeVCluster     = register(IsolationModeVClusterName, isolationModeVClusterYAML)
)
