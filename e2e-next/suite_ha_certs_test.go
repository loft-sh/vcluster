// Suite: certs-vcluster
// vCluster: CertsVCluster (dedicated single-replica with deploy etcd)
// Run:      just run-e2e 'certs'
//
// All cert tests run in a single Ordered Describe because:
//   - Cert rotation restarts the vcluster pod, killing any shared proxy
//   - The expiration test uses os.Setenv(VCLUSTER_CERTS_VALIDITYPERIOD) which is
//     process-global and would poison parallel cert operations
//   - Each section's reconnect establishes the proxy for the next section
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_security/certs"
	. "github.com/onsi/ginkgo/v2"
)

func init() { suiteCertsVCluster() }

func suiteCertsVCluster() {
	// Ordered: CertTestsSpec must complete before CertAutoRotationSpec because
	// the auto-rotation test patches the cert secret and triggers pod restarts,
	// which causes the vcluster to briefly enter "Terminating" status.
	Describe("certs-vcluster",
		Ordered,
		cluster.Use(clusters.CertsVCluster),
		cluster.Use(clusters.HostCluster),
		func() {
			certs.CertTestsSpec()
			certs.CertAutoRotationSpec()
		},
	)
}
