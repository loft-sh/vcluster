// Suite: e2e_certs
// Matches: test/e2e_certs/certs/rotate.go (all cert rotation scenarios)
// vCluster: CertsVCluster (dedicated single-replica with deploy etcd)
// Run:      just run-e2e 'security && !non-default'
//
// All cert tests run in a single Ordered Describe because:
// - Cert rotation restarts the vcluster pod, killing any shared proxy
// - The expiration test uses os.Setenv(VCLUSTER_CERTS_VALIDITYPERIOD) which is
//   process-global and would poison parallel cert operations
// - Each section's reconnect establishes the proxy for the next section
package e2e_next

import (
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/certs"
)

var _ = certs.DescribeCertTests(clusters.CertsVCluster)
