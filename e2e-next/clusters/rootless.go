package clusters

import _ "embed"

// RootlessVCluster runs as non-root (runAsUser: 12345, fsGroup: 12345)
// for rootless mode verification tests.

//go:embed vcluster-rootless.yaml
var rootlessVClusterYAML string

var (
	RootlessVClusterName = "rootless-vcluster"
	RootlessVCluster     = register(RootlessVClusterName, rootlessVClusterYAML)
)
