// Suite: e2e_node
// Matches: test/e2e_node/e2e_node_suite_test.go
// Run:      just run-e2e 'node-sync-vcluster && !non-default'
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/coredns"
	"github.com/loft-sh/vcluster/e2e-next/test_core/nodesync"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	"github.com/loft-sh/vcluster/e2e-next/test_core/webhook"
	. "github.com/onsi/ginkgo/v2"
)

func init() {
	suiteNodeSyncVCluster()
}

func suiteNodeSyncVCluster() {
	Describe("node-sync-vcluster",
		cluster.Use(clusters.NodeSyncVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			nodesync.NodeSyncSpec()
			coredns.CoreDNSSpec()
			test_core.PodSyncSpec()
			test_core.PVCSyncSpec()
			test_core.NetworkPolicyEnforcementSpec()
			webhook.AdmissionWebhookSpec()
		},
	)
}
