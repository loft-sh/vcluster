// Suite: cli-vcluster
// vCluster: CLIVCluster (dedicated instance for CLI connect tests)
// Run:      just run-e2e 'cli'
// Separate from CommonVCluster because connect operations create port-forward
// processes that can disrupt the shared background proxy used by sync tests.
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/lifecycle"
	. "github.com/onsi/ginkgo/v2"
)

func init() {
	suiteCLIVCluster()
}

func suiteCLIVCluster() {
	Describe("cli-vcluster",
		cluster.Use(clusters.CLIVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			lifecycle.ConnectSpec()
		},
	)
}
