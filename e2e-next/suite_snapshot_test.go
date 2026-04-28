// Suite: snapshot-vcluster
// vCluster: CSI hostpath driver + snapshot CRDs. PreSetup installs CSI
// hostpath driver and creates snapshot-data PVC. Lifecycle owned by this
// Describe's BeforeAll + DeferCleanup.
// Run:      just run-e2e 'snapshots'
package e2e_next

import (
	"context"
	_ "embed"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/e2e-next/setup"
	"github.com/loft-sh/vcluster/e2e-next/setup/lazyvcluster"
	"github.com/loft-sh/vcluster/e2e-next/test_storage/snapshot"
	. "github.com/onsi/ginkgo/v2"
)

//go:embed vcluster-snapshot.yaml
var snapshotVClusterYAML string

const snapshotVClusterName = "snapshot-vcluster"

func init() { suiteSnapshotVCluster() }

func suiteSnapshotVCluster() {
	Describe("snapshot-vcluster", labels.Snapshots, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx,
					snapshotVClusterName,
					snapshotVClusterYAML,
					lazyvcluster.WithPreSetup(setup.SnapshotPreSetup(snapshotVClusterName)),
				)
			})

			snapshot.SnapshotAllSpec()
		},
	)
}
