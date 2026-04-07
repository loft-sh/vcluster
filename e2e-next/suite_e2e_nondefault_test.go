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
