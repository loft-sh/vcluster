// Suite: common-vcluster (main PR-gating tests)
// vCluster: comprehensive config with all sync options enabled.
// Lifecycle owned by this Describe's BeforeAll + DeferCleanup.
// Run:      just run-e2e 'pr && !non-default'
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
	"github.com/loft-sh/vcluster/e2e-next/test_core/sync/fromhost"
	"github.com/loft-sh/vcluster/e2e-next/test_deploy"
	"github.com/loft-sh/vcluster/e2e-next/test_security/webhook"
	. "github.com/onsi/ginkgo/v2"
)

//go:embed vcluster-default.yaml
var commonVClusterYAML string

const commonVClusterName = "common-vcluster"

func init() { suiteCommonVCluster() }

// Ordered: the outer Describe owns vCluster lifecycle via BeforeAll +
// DeferCleanup - Ginkgo only allows BeforeAll/AfterAll inside Ordered
// containers.
func suiteCommonVCluster() {
	Describe("common-vcluster", labels.PR, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx, commonVClusterName, commonVClusterYAML)
			})

			test_core.PodSyncSpec()
			test_core.NetworkPolicySyncSpec()
			test_core.PVCSyncSpec()
			test_core.K8sDefaultEndpointSpec()
			test_core.NodeSyncLabelSelectorSpec()
			test_core.ServiceBasicSyncSpec()
			coredns.CoreDNSSpec()
			webhook.AdmissionWebhookSpec()
			fromhost.EventSyncSpec()
			fromhost.FromHostConfigMapsSpec()
			fromhost.FromHostSecretsSpec()
			test_deploy.HelmChartsSpec()
			test_deploy.InitManifestsSpec()
		},
	)
}
