// Suite: migration-vcluster
// Tests upgrade and migration flows that need to start from an older released
// vCluster chart before upgrading to the current local chart.
// Run: just run-e2e 'migration'
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	testmigration "github.com/loft-sh/vcluster/e2e-next/test_migration"
	. "github.com/onsi/ginkgo/v2"
)

func init() {
	suiteMigrationVCluster()
}

func suiteMigrationVCluster() {
	Describe("migration-vcluster",
		labels.Migration,
		Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			testmigration.K3SToK8SMigrationSpec()
		},
	)
}
