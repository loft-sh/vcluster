// Suite: e2e_cert_rotation
// Tests serving cert hot-reload and auto-rotation with short-lived certs.
// vCluster: ShortCertsVCluster (DEVELOPMENT=true, 3m cert validity)
// Run:      just run-e2e '/short-certs-vcluster/ && !non-default'
package e2e_next

import (
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/certs"
)

var (
	_ = certs.DescribeServingCertRotation(clusters.ShortCertsVCluster)
)
