// Suite: fromhost-limitclasses-vcluster
// vCluster: FromHostLimitClassesVCluster (fromHost sync with label-selector limits)
// Run:      just run-e2e 'pr && ingressclasses'
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/e2e-next/test_core/sync/fromhost"
	. "github.com/onsi/ginkgo/v2"
)

func init() { suiteFromHostLimitClassesVCluster() }

func suiteFromHostLimitClassesVCluster() {
	Describe("fromhost-limitclasses-vcluster", labels.PR,
		cluster.Use(clusters.FromHostLimitClassesVCluster),
		func() {
			fromhost.FromHostIngressClassesSpec()
			fromhost.FromHostStorageClassesSpec()
			fromhost.FromHostPriorityClassesSpec()
			fromhost.FromHostRuntimeClassesSpec()
		},
	)
}
