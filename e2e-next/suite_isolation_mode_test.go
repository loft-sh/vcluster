// Suite: isolation-mode-vcluster
// vCluster: podSecurityStandard, resourceQuota, limitRange.
// Run:      just run-e2e 'isolation'
package e2e_next

import (
	"context"
	_ "embed"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/e2e-next/setup/lazyvcluster"
	"github.com/loft-sh/vcluster/e2e-next/test_core/coredns"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	"github.com/loft-sh/vcluster/e2e-next/test_security/isolation"
	"github.com/loft-sh/vcluster/e2e-next/test_security/webhook"
	. "github.com/onsi/ginkgo/v2"
)

//go:embed vcluster-isolation-mode.yaml
var isolationModeVClusterYAML string

const isolationModeVClusterName = "isolation-mode-vcluster"

func init() { suiteIsolationModeVCluster() }

func suiteIsolationModeVCluster() {
	Describe("isolation-mode-vcluster", labels.Isolation, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx, isolationModeVClusterName, isolationModeVClusterYAML)
			})

			isolation.IsolationModeSpec()
			coredns.CoreDNSSpec()
			test_core.PodSyncSpec()
			test_core.PVCSyncSpec()
			webhook.AdmissionWebhookSpec()
		},
	)
}
