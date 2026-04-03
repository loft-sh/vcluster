// Suite: snapshot-vcluster
// vCluster: SnapshotVCluster (CSI hostpath driver + snapshot CRDs)
// Run:      just run-e2e 'snapshot-vcluster'
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/snapshot"
	. "github.com/onsi/ginkgo/v2"
)

func init() {
	suiteSnapshotVCluster()
}

func suiteSnapshotVCluster() {
	Describe("snapshot-vcluster",
		cluster.Use(clusters.SnapshotVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			snapshot.SnapshotAllSpec()
		},
	)
}
