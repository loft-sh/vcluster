// Suite: e2e_ha_cert_rotation
// Tests HA-coordinated cert rotation with lease-based coordination.
// vCluster: HAShortCertsVCluster (2 replicas, 3m cert validity, 15s check interval)
// Run:      just run-e2e 'ha-short-certs-vcluster'
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_security/certs"
	. "github.com/onsi/ginkgo/v2"
)

func init() {
	suiteHAShortCertsVCluster()
}

func suiteHAShortCertsVCluster() {
	Describe("ha-short-certs-vcluster",
		cluster.Use(clusters.HAShortCertsVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			certs.HACertRotationSpec()
		},
	)
}
