// Suite: e2e_ha_certs
// Matches: test/e2e_ha/e2e_suite_test.go (cert rotation only)
// vCluster: HAVCluster (3 replicas, HA etcd + coredns)
// Run:      just run-e2e '/ha-certs-vcluster/ && !non-default'
package e2e_next

import (
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/certs"
)

var (
	_ = certs.DescribeCertRotation(clusters.HAVCluster)
)
