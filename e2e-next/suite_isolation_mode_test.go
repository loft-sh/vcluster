package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/coredns"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	"github.com/loft-sh/vcluster/e2e-next/test_security/isolation"
	"github.com/loft-sh/vcluster/e2e-next/test_security/webhook"
	. "github.com/onsi/ginkgo/v2"
)

func init() { suiteIsolationModeVCluster() }

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
