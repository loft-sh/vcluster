// Suite: kubelet-proxy-vcluster
// vCluster: KubeletProxyVCluster (kubelet proxy with restricted subpaths)
// Run:      just run-e2e 'pr && kubelet-proxy-vcluster'
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	test_core "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	. "github.com/onsi/ginkgo/v2"
)

func init() {
	suiteKubeletProxyVCluster()
}

func suiteKubeletProxyVCluster() {
	Describe("kubelet-proxy-vcluster", labels.PR,
		cluster.Use(clusters.KubeletProxyVCluster),
		func() {
			test_core.KubeletProxySpec()
		},
	)
}
