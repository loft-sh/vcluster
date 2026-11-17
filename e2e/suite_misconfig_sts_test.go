// Suite: misconfig-sts-vcluster
// vCluster: default config, embedded SQLite on a volume claim per pod (StatefulSet).
// Run:      just run-e2e 'misconfig'
package e2e

import (
	"context"
	_ "embed"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e/clusters"
	"github.com/loft-sh/vcluster/e2e/labels"
	"github.com/loft-sh/vcluster/e2e/setup/lazyvcluster"
	"github.com/loft-sh/vcluster/e2e/test_core/misconfig"
	. "github.com/onsi/ginkgo/v2"
)

//go:embed vcluster-misconfig-sts.yaml
var misconfigStsVClusterYAML string

const misconfigStsVClusterName = "misconfig-sts-vcluster"

func init() { suiteMisconfigStsVCluster() }

// Ordered because the spec scales the control plane and keeps state across steps.
func suiteMisconfigStsVCluster() {
	Describe("misconfig-sts-vcluster", labels.Misconfig, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx, misconfigStsVClusterName, misconfigStsVClusterYAML)
			})

			misconfig.ReplicaMismatchSpec(misconfig.StatefulSet)
		},
	)
}
