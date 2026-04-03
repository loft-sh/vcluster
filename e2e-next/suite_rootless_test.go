// Suite: e2e_rootless
// Matches: test/e2e_rootless/e2e_rootless_mode_suite_test.go
// Run:      just run-e2e 'rootless-vcluster && !non-default'
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/coredns"
	"github.com/loft-sh/vcluster/e2e-next/test_core/rootless"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	"github.com/loft-sh/vcluster/e2e-next/test_core/webhook"
	. "github.com/onsi/ginkgo/v2"
)

func init() {
	suiteRootlessVCluster()
}

func suiteRootlessVCluster() {
	Describe("rootless-vcluster",
		cluster.Use(clusters.RootlessVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			rootless.RootlessModeSpec()
			coredns.CoreDNSSpec()
			test_core.PodSyncSpec()
			test_core.PVCSyncSpec()
			test_core.NetworkPolicyEnforcementSpec()
			webhook.AdmissionWebhookSpec()
		},
	)
}
