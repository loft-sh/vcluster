// Suite: service-sync-vcluster
// vCluster: networking.replicateServices config.
// Run:      just run-e2e 'pr && sync'
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

//go:embed vcluster-servicesync.yaml
var serviceSyncVClusterYAML string

const serviceSyncVClusterName = "service-sync-vcluster"

func init() { suiteServiceSyncVCluster() }

func suiteServiceSyncVCluster() {
	Describe("service-sync-vcluster", labels.PR, labels.Sync, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx, serviceSyncVClusterName, serviceSyncVClusterYAML)
			})

			test_core.ServiceSyncSpec()
		},
	)
}
