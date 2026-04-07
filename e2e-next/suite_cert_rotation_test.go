// Suite: e2e_serving_cert_rotation
// Tests serving cert hot-reload with short-lived certs (syncer SANs bug fix).
// vCluster: ShortCertsVCluster (DEVELOPMENT=true, 3m cert validity)
// Run:      just run-e2e 'short-certs-vcluster'
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/certs"
	. "github.com/onsi/ginkgo/v2"
)

func init() {
	suiteShortCertsVCluster()
}

func suiteShortCertsVCluster() {
	Describe("short-certs-vcluster",
		cluster.Use(clusters.ShortCertsVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			certs.ServingCertRotationSpec()
		},
	)
}
