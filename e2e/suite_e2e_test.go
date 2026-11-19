// Suite: common-vcluster (main PR-gating tests)
// vCluster: comprehensive config with all sync options enabled.
// Run:      just run-e2e 'pr && !non-default'
package e2e

import (
	"context"
	_ "embed"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e/clusters"
	"github.com/loft-sh/vcluster/e2e/labels"
	"github.com/loft-sh/vcluster/e2e/setup"
	"github.com/loft-sh/vcluster/e2e/setup/lazyvcluster"
	"github.com/loft-sh/vcluster/e2e/test_core/coredns"
	"github.com/loft-sh/vcluster/e2e/test_core/metrics"
	test_core "github.com/loft-sh/vcluster/e2e/test_core/sync"
	"github.com/loft-sh/vcluster/e2e/test_core/sync/fromhost"
	"github.com/loft-sh/vcluster/e2e/test_deploy"
	"github.com/loft-sh/vcluster/e2e/test_security/webhook"
	. "github.com/onsi/ginkgo/v2"
)

//go:embed vcluster-default.yaml
var commonVClusterYAML string

const commonVClusterName = "common-vcluster"

func init() { suiteCommonVCluster() }

func suiteCommonVCluster() {
	Describe("common-vcluster", labels.PR, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				// The namespaces mapped by fromHost configmap and secret sync in vcluster-default.yaml.
				return lazyvcluster.LazyVCluster(ctx, commonVClusterName, commonVClusterYAML,
					lazyvcluster.WithPreSetup(setup.HostNamespacesPreSetup(
						"from-host-sync-test",
						"from-host-sync-test-2",
						"from-host-sync-test-3",
						"my-ns",
						"same-ns",
					)),
				)
			})

			test_core.PodSyncSpec()
			test_core.SyncErrorSanitisationSpec()
			test_core.HostNameCollisionSpec()
			test_core.NetworkPolicySyncSpec()
			test_core.PVCSyncSpec()
			test_core.K8sDefaultEndpointSpec()
			test_core.NodeSyncLabelSelectorSpec()
			test_core.ServiceBasicSyncSpec()
			coredns.CoreDNSSpec()
			metrics.SyncerMetricsSpec()
			webhook.AdmissionWebhookSpec()
			fromhost.EventSyncSpec()
			fromhost.FromHostConfigMapsSpec()
			fromhost.FromHostSecretsSpec()
			test_deploy.HelmChartsSpec()
			test_deploy.InitManifestsSpec()
		},
	)
}
