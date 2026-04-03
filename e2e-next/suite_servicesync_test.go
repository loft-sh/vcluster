// Suite: service-sync-vcluster
// vCluster: ServiceSyncVCluster (replicateServices config)
// Run:      just run-e2e 'pr && service-sync-vcluster'
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	. "github.com/onsi/ginkgo/v2"
)

func init() {
	suiteServiceSyncVCluster()
}

func suiteServiceSyncVCluster() {
	Describe("service-sync-vcluster", labels.PR,
		cluster.Use(clusters.ServiceSyncVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			test_core.ServiceSyncSpec()
		},
	)
}
