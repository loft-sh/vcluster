// Suite: e2e_isolation_mode
// Matches: test/e2e_isolation_mode/e2e_isolation_mode_test.go
// vCluster: IsolationModeVCluster (podSecurityStandard, resourceQuota, limitRange)
// Run:      just run-e2e '/isolation-mode/ && !non-default'
package e2e_next

import (
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/coredns"
	"github.com/loft-sh/vcluster/e2e-next/test_core/isolation"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	"github.com/loft-sh/vcluster/e2e-next/test_core/webhook"
)

var (
	_ = isolation.DescribeIsolationMode(clusters.IsolationModeVCluster)
	_ = coredns.DescribeCoreDNS(clusters.IsolationModeVCluster)
	_ = test_core.DescribePodSync(clusters.IsolationModeVCluster)
	_ = test_core.DescribePVCSync(clusters.IsolationModeVCluster)
	_ = test_core.DescribeNetworkPolicyEnforcement(clusters.IsolationModeVCluster)
	_ = webhook.DescribeAdmissionWebhook(clusters.IsolationModeVCluster)
)
