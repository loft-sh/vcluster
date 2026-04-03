// Suite: e2e_isolation_mode
// Matches: test/e2e_isolation_mode/e2e_isolation_mode_test.go
// Run:      just run-e2e 'isolation-mode-vcluster && !non-default'
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/coredns"
	"github.com/loft-sh/vcluster/e2e-next/test_core/isolation"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	"github.com/loft-sh/vcluster/e2e-next/test_core/webhook"
	. "github.com/onsi/ginkgo/v2"
)

func init() {
	suiteIsolationModeVCluster()
}

func suiteIsolationModeVCluster() {
	Describe("isolation-mode-vcluster",
		cluster.Use(clusters.IsolationModeVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			isolation.IsolationModeSpec()
			coredns.CoreDNSSpec()
			test_core.PodSyncSpec()
			test_core.PVCSyncSpec()
			test_core.NetworkPolicyEnforcementSpec()
			webhook.AdmissionWebhookSpec()
		},
	)
}
