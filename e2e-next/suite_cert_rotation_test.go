// Suite: e2e_short_certs
// Tests serving cert hot-reload (syncer SANs bug fix) and single-replica
// watcher rotation with short-lived certs, including rollout propagation.
// vCluster: ShortCertsVCluster (DEVELOPMENT=true, 3m cert validity, 15s check interval)
// Run:      just run-e2e 'short-certs-vcluster'
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_security/certs"
	. "github.com/onsi/ginkgo/v2"
)

func init() {
	suiteShortCertsVCluster()
}

func suiteShortCertsVCluster() {
	// Ordered: ServingCertRotationSpec must complete before
	// SingleReplicaWatcherSpec because the watcher test triggers a workload
	// rollout which would disrupt the serving cert Consistently check.
	Describe("short-certs-vcluster",
		Ordered,
		cluster.Use(clusters.ShortCertsVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			certs.ServingCertRotationSpec()
			certs.SingleReplicaWatcherSpec()
		},
	)
}
