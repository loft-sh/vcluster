// Suite: kubelet-proxy-vcluster
// vCluster: kubelet proxy with restricted subpaths.
// Lifecycle owned by this Describe's BeforeAll + DeferCleanup.
// Run:      just run-e2e 'security'
package e2e_next

import (
	"context"
	_ "embed"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/e2e-next/setup/lazyvcluster"
	"github.com/loft-sh/vcluster/e2e-next/test_security/kubeletproxy"
	. "github.com/onsi/ginkgo/v2"
)

//go:embed vcluster-kubelet-proxy.yaml
var kubeletProxyVClusterYAML string

const kubeletProxyVClusterName = "kubelet-proxy-vcluster"

func init() { suiteKubeletProxyVCluster() }

// Ordered: the outer Describe owns vCluster lifecycle via BeforeAll +
// DeferCleanup - Ginkgo only allows BeforeAll/AfterAll inside Ordered
// containers.
func suiteKubeletProxyVCluster() {
	Describe("kubelet-proxy-vcluster", labels.PR, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx, kubeletProxyVClusterName, kubeletProxyVClusterYAML)
			})

			kubeletproxy.KubeletProxySpec()
		},
	)
}
