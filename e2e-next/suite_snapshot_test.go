// Suite: snapshot-vcluster
// vCluster: SnapshotVCluster (CSI hostpath driver + snapshot CRDs)
// PreSetup: installs CSI hostpath driver and creates snapshot-data PVC
// Run:      just run-e2e 'snapshots'
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_storage/snapshot"
	. "github.com/onsi/ginkgo/v2"
)

func init() { suiteSnapshotVCluster() }

func suiteSnapshotVCluster() {
	Describe("snapshot-vcluster",
		cluster.Use(clusters.SnapshotVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			snapshot.SnapshotAllSpec()
		},
	)
}
