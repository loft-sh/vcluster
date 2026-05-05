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

const snapshotLargeRestoreVClusterName = "large-restore-vcluster"

func init() { suiteSnapshotLargeRestore() }

func suiteSnapshotLargeRestore() {
	Describe("large-restore-vcluster", labels.SnapshotLargeRestore, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx,
					snapshotLargeRestoreVClusterName,
					snapshotVClusterYAML,
					lazyvcluster.WithPreSetup(setup.SnapshotPreSetup(snapshotLargeRestoreVClusterName)),
				)
			})

			snapshot.SnapshotLargeRestoreSpec()
		},
	)
}
