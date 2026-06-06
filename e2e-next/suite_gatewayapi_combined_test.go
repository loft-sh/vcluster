// Suite: gatewayapi-combined-vcluster
// vCluster: fromHost Gateways and toHost Gateway API Gateways enabled together.
// Run:      just run-e2e 'pr && gatewayapi'
package e2e_next

import (
	"context"
	_ "embed"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/e2e-next/setup"
	"github.com/loft-sh/vcluster/e2e-next/setup/lazyvcluster"
	"github.com/loft-sh/vcluster/e2e-next/test_gatewayapi"
	. "github.com/onsi/ginkgo/v2"
)

//go:embed vcluster-gatewayapi-combined.yaml
var gatewayAPICombinedVClusterYAML string

const gatewayAPICombinedVClusterName = "gatewayapi-combined-vcluster"

func init() { suiteGatewayAPICombinedVCluster() }

func suiteGatewayAPICombinedVCluster() {
	// Ordered so the startup regression coverage runs against one vCluster with
	// both Gateway import and tenant Gateway sync enabled.
	Describe("gatewayapi-combined-vcluster", labels.PR, labels.GatewayAPI, labels.GatewayClasses, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx,
					gatewayAPICombinedVClusterName,
					gatewayAPICombinedVClusterYAML,
					lazyvcluster.WithPreSetup(setup.GatewayAPIPreSetup()),
				)
			})

			test_gatewayapi.GatewayAPICombinedSpec()
		},
	)
}
