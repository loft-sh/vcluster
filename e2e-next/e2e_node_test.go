// Suite: e2e_node
// Matches: test/e2e_node/e2e_node_suite_test.go
// vCluster: NodeSyncVCluster (virtualScheduler, all host nodes synced)
// Run:      just run-e2e '/node-sync-vcluster/'
package e2e_next

import (
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/coredns"
	"github.com/loft-sh/vcluster/e2e-next/test_core/nodesync"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	"github.com/loft-sh/vcluster/e2e-next/test_core/webhook"
)

var (
	_ = nodesync.DescribeNodeSync(clusters.NodeSyncVCluster)
	_ = coredns.DescribeCoreDNS(clusters.NodeSyncVCluster)
	_ = test_core.DescribePodSync(clusters.NodeSyncVCluster)
	_ = test_core.DescribePVCSync(clusters.NodeSyncVCluster)
	_ = test_core.DescribeNetworkPolicyEnforcement(clusters.NodeSyncVCluster)
	_ = webhook.DescribeAdmissionWebhook(clusters.NodeSyncVCluster)
)
