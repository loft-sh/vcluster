// Suite: lifecycle-vcluster
// vCluster: CommonVCluster (reuses the common config for CLI connect tests)
// Connect tests run against a dedicated vCluster instance to avoid disrupting
// the shared background proxy used by other suites.
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/lifecycle"
	. "github.com/onsi/ginkgo/v2"
)

func init() {
	suiteLifecycleVCluster()
}

func suiteLifecycleVCluster() {
	Describe("lifecycle-vcluster",
		cluster.Use(clusters.CommonVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			lifecycle.ConnectSpec()
		},
	)
}
