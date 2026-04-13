// Suite: cli-vcluster
// vCluster: CLIVCluster (dedicated instance for CLI connect tests)
// Run:      just run-e2e 'cli'
// Prereq:   vcluster binary must be in $PATH
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
	// Tenant cluster lifecycle tests create their own vclusters via CLI
	// (no cluster.Use dependency needed).
	lifecycle.TenantClusterLifecycleSpec()
	lifecycle.PauseResumeScaledDownSpec()
}

// Ordered because PauseResumeSpec is destructive - it kills the vcluster pods
// and background proxy. ConnectSpec must run first while the vcluster is healthy.
func suiteCLIVCluster() {
	Describe("cli-vcluster", Ordered,
		cluster.Use(clusters.CLIVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			lifecycle.ConnectSpec()
			lifecycle.PauseResumeSpec()
		},
	)
}
