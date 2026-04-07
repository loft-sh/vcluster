// Suite: kubelet-proxy-vcluster
// vCluster: KubeletProxyVCluster (kubelet proxy with restricted subpaths)
// Run:      just run-e2e 'security'
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/e2e-next/test_security/kubeletproxy"
	. "github.com/onsi/ginkgo/v2"
)

func init() { suiteKubeletProxyVCluster() }

func suiteKubeletProxyVCluster() {
	Describe("kubelet-proxy-vcluster", labels.PR,
		cluster.Use(clusters.KubeletProxyVCluster),
		func() {
			kubeletproxy.KubeletProxySpec()
		},
	)
}
