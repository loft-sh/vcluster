// Suite: versionskew-vcluster
// vCluster: virtual cluster pinned below K8s 1.34 on a host at 1.34+, to cover
//
//	the version skew handling of the pod syncer (issue #3578).
//
// Run:      just run-e2e 'sync'
package e2e_next

import (
	"context"
	_ "embed"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/e2e-next/setup/lazyvcluster"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	. "github.com/onsi/ginkgo/v2"
)

//go:embed vcluster-versionskew.yaml
var versionSkewVClusterYAML string

const versionSkewVClusterName = "versionskew-vcluster"

func init() { suiteVersionSkewVCluster() }

func suiteVersionSkewVCluster() {
	Describe("versionskew-vcluster", labels.PR, labels.Sync, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx, versionSkewVClusterName, versionSkewVClusterYAML)
			})

			test_core.PodVersionSkewSpec()
		},
	)
}
