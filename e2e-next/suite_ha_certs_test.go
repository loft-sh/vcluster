// Suite: e2e_certs
// Matches: test/e2e_certs/certs/rotate.go (all cert rotation scenarios)
// vCluster: CertsVCluster (dedicated single-replica with deploy etcd)
// Run:      just run-e2e 'security && !non-default'
//
// Cert rotation restarts the vcluster pod, killing the proxy for any tests
// sharing the same cluster. That's why these tests use a dedicated CertsVCluster
// rather than CommonVCluster.
package e2e_next

import (
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/certs"
)

var (
	_ = certs.DescribeCertRotation(clusters.CertsVCluster)
	_ = certs.DescribeCertExpiration(clusters.CertsVCluster)
	_ = certs.DescribeCertKubeConfig(clusters.CertsVCluster)
)
