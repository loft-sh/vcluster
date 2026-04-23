// Suite: cli-vcluster
// vCluster: dedicated instance for CLI connect tests.
// Run:      just run-e2e 'cli'
// Prereq:   vcluster binary must be in $PATH
// Separate from CommonVCluster because connect operations create port-forward
// processes that can disrupt the shared background proxy used by sync tests.
package e2e_next

import (
	"context"
	_ "embed"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/e2e-next/setup/lazyvcluster"
	"github.com/loft-sh/vcluster/e2e-next/test_core/lifecycle"
	. "github.com/onsi/ginkgo/v2"
)

//go:embed vcluster-cli.yaml
var cliVClusterYAML string

const cliVClusterName = "cli-vcluster"

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
	Describe("cli-vcluster", labels.CLI, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx, cliVClusterName, cliVClusterYAML)
			})

			lifecycle.ConnectSpec()
			lifecycle.PauseResumeSpec()
		},
	)
}
