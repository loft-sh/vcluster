// Suite: volumesnapshot-vcluster
// vCluster: shared nodes with volumeSnapshots, volumeSnapshotContents and
// volumeSnapshotClasses sync enabled. PreSetup installs the CSI hostpath
// driver, the snapshot CRDs and the snapshot-controller on the host.
// Run:      just run-e2e 'volumesnapshots'
package e2e

import (
	"context"
	_ "embed"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e/clusters"
	"github.com/loft-sh/vcluster/e2e/labels"
	"github.com/loft-sh/vcluster/e2e/setup"
	"github.com/loft-sh/vcluster/e2e/setup/lazyvcluster"
	"github.com/loft-sh/vcluster/e2e/test_storage/volumesnapshot"
	. "github.com/onsi/ginkgo/v2"
)

//go:embed vcluster-volumesnapshot.yaml
var volumeSnapshotVClusterYAML string

const volumeSnapshotVClusterName = "volumesnapshot-vcluster"

func init() { suiteVolumeSnapshotVCluster() }

func suiteVolumeSnapshotVCluster() {
	Describe("volumesnapshot-vcluster", labels.VolumeSnapshots, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx,
					volumeSnapshotVClusterName,
					volumeSnapshotVClusterYAML,
					lazyvcluster.WithPreSetup(setup.CSIHostpathPreSetup()),
				)
			})

			volumesnapshot.VolumeSnapshotSyncSpec()
		},
	)
}
