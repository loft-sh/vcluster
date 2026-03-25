package clusters

import _ "embed"

// NodeSyncVCluster enables virtualScheduler and syncs ALL host nodes
// (selector.all: true) for node-specific tests.

//go:embed vcluster-node.yaml
var nodeSyncVClusterYAML string

var (
	NodeSyncVClusterName = "node-sync-vcluster"
	NodeSyncVCluster     = register(NodeSyncVClusterName, nodeSyncVClusterYAML)
)
