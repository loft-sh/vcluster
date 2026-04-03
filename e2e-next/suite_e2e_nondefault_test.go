// Suite: common-vcluster-non-default
// vCluster: CommonVCluster (same cluster, but tests require special infra like Calico CNI)
// Run:      just run-e2e 'non-default'
//
// Separated from suite_e2e_test.go to avoid inheriting labels.PR, which would
// cause the CI label filter "!non-default || pr" to run these on every PR.
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	. "github.com/onsi/ginkgo/v2"
)

func init() {
	suiteCommonVClusterNonDefault()
}

func suiteCommonVClusterNonDefault() {
	Describe("common-vcluster-non-default",
		cluster.Use(clusters.CommonVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			test_core.NetworkPolicyEnforcementSpec()
		},
	)
}
