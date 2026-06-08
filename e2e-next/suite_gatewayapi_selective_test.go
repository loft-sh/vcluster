// Suite: gatewayapi-selective-vcluster
// vCluster: broken-out toHost.gatewayApi sub-toggles coverage (TC-02a variant B).
// Only gateways are enabled; httpRoutes and referenceGrants are off.
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

//go:embed vcluster-gatewayapi-selective.yaml
var gatewayAPISelectiveVClusterYAML string

const gatewayAPISelectiveVClusterName = "gatewayapi-selective-vcluster"

func init() { suiteGatewayAPISelectiveVCluster() }

func suiteGatewayAPISelectiveVCluster() {
	Describe("gatewayapi-selective-vcluster", labels.PR, labels.GatewayAPI, labels.GatewayClasses, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx,
					gatewayAPISelectiveVClusterName,
					gatewayAPISelectiveVClusterYAML,
					lazyvcluster.WithPreSetup(setup.GatewayAPIPreSetup()),
				)
			})

			test_gatewayapi.GatewayAPISelectiveSpec()
		},
	)
}
