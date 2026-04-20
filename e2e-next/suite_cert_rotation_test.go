// Suite: short-certs-vcluster
// Tests serving cert hot-reload (syncer SANs bug fix) and single-replica
// watcher rotation with short-lived certs, including rollout propagation.
// vCluster: DEVELOPMENT=true, 3m cert validity, 15s check interval.
// Lifecycle owned by this Describe's BeforeAll + DeferCleanup.
// Run:      just run-e2e 'short-certs-vcluster'
package e2e_next

import (
	"context"
	_ "embed"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/setup/lazyvcluster"
	"github.com/loft-sh/vcluster/e2e-next/test_security/certs"
	. "github.com/onsi/ginkgo/v2"
)

//go:embed vcluster-short-certs.yaml
var shortCertsVClusterYAML string

const shortCertsVClusterName = "short-certs-vcluster"

func init() {
	suiteShortCertsVCluster()
}

func suiteShortCertsVCluster() {
	// Ordered: ServingCertRotationSpec must complete before
	// SingleReplicaWatcherSpec because the watcher test triggers a workload
	// rollout which would disrupt the serving cert Consistently check.
	Describe("short-certs-vcluster",
		Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx, shortCertsVClusterName, shortCertsVClusterYAML)
			})

			certs.ServingCertRotationSpec()
			certs.SingleReplicaWatcherSpec()
		},
	)
}
