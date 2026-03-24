// Suite: e2e_rootless
// Matches: test/e2e_rootless/e2e_rootless_mode_suite_test.go
// vCluster: RootlessVCluster (runAsUser: 12345, fsGroup: 12345)
// Run:      just run-e2e '/rootless-vcluster/ && !non-default'
package e2e_next

import (
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/coredns"
	"github.com/loft-sh/vcluster/e2e-next/test_core/rootless"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	"github.com/loft-sh/vcluster/e2e-next/test_core/webhook"
)

var (
	_ = rootless.DescribeRootlessMode(clusters.RootlessVCluster)
	_ = coredns.DescribeCoreDNS(clusters.RootlessVCluster)
	_ = test_core.DescribePodSync(clusters.RootlessVCluster)
	_ = test_core.DescribePVCSync(clusters.RootlessVCluster)
	_ = test_core.DescribeNetworkPolicyEnforcement(clusters.RootlessVCluster)
	_ = webhook.DescribeAdmissionWebhook(clusters.RootlessVCluster)
)
