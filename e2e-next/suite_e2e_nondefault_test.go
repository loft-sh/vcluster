// Suite: common-vcluster-non-default
// vCluster: CommonVCluster (same cluster as suite_e2e, but tests require special infra)
// Separated from the PR suite to avoid inheriting labels.PR, which would cause
// CI label filter "!non-default || pr" to include these tests on PRs.
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	. "github.com/onsi/ginkgo/v2"
)

func init() { suiteCommonVClusterNonDefault() }

func suiteCommonVClusterNonDefault() {
	Describe("common-vcluster-non-default",
		cluster.Use(clusters.CommonVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			test_core.NetworkPolicyEnforcementSpec()
		},
	)
}
