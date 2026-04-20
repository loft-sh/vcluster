// Suite: fromhost-limitclasses-vcluster
// vCluster: fromHost sync with label-selector limits (lifecycle owned by
// the parent Describe via plain Ginkgo BeforeAll + DeferCleanup).
// Run:      just run-e2e 'pr && ingressclasses'
package e2e_next

import (
	"context"
	_ "embed"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/e2e-next/setup/lazyvcluster"
	"github.com/loft-sh/vcluster/e2e-next/test_core/sync/fromhost"
	. "github.com/onsi/ginkgo/v2"
)

//go:embed vcluster-fromhost-limitclasses.yaml
var fromHostLimitClassesVClusterYAML string

const fromHostLimitClassesVClusterName = "fromhost-limitclasses-vcluster"

func init() { suiteFromHostLimitClassesVCluster() }

// suiteFromHostLimitClassesVCluster wraps the Describe in a named function
// so the framework's package-init safety net allows the
// cluster.Use(HostCluster) dependency.
//
// Ordered: the outer Describe owns vCluster lifecycle via BeforeAll +
// DeferCleanup - Ginkgo only allows BeforeAll/AfterAll inside Ordered
// containers.
func suiteFromHostLimitClassesVCluster() {
	Describe("fromhost-limitclasses-vcluster", labels.PR, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx,
					fromHostLimitClassesVClusterName,
					fromHostLimitClassesVClusterYAML,
				)
			})

			fromhost.FromHostIngressClassesSpec()
			fromhost.FromHostStorageClassesSpec()
			fromhost.FromHostPriorityClassesSpec()
			fromhost.FromHostRuntimeClassesSpec()
		},
	)
}
