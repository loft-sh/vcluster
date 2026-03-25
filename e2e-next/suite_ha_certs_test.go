// Suite: e2e_ha_certs
// Matches: test/e2e_ha/e2e_suite_test.go (cert rotation only)
// vCluster: HAVCluster (3 replicas, HA etcd + coredns)
// Run:      just run-e2e '/ha-certs-vcluster/ && !non-default'
//
// NOTE: The old e2e_ha suite only ran cert rotation tests. Broader HA functional
// tests (pod/service/DNS behavior under HA) were not part of the old suite and
// are not registered here. Add them as needed when HA-specific behavior tests
// are written.
package e2e_next

import (
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/test_core/certs"
)

var (
	_ = certs.DescribeCertRotation(clusters.HAVCluster)
)
