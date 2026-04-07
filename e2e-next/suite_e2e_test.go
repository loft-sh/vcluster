// Suite: common-vcluster (main PR-gating tests)
// vCluster: CommonVCluster (comprehensive config with all sync options enabled)
// Run:      just run-e2e 'pr && !non-default'
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/e2e-next/test_core/coredns"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	"github.com/loft-sh/vcluster/e2e-next/test_core/sync/fromhost"
	"github.com/loft-sh/vcluster/e2e-next/test_deploy"
	"github.com/loft-sh/vcluster/e2e-next/test_security/webhook"
	. "github.com/onsi/ginkgo/v2"
)

func init() { suiteCommonVCluster() }

func suiteCommonVCluster() {
	Describe("common-vcluster", labels.PR,
		cluster.Use(clusters.CommonVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
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
