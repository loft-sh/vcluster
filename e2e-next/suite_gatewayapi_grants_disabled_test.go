// Suite: gatewayapi-grants-disabled-vcluster
// vCluster: Gateway API route sync with referenceGrants.enabled "false" —
// route controllers must start and virtual ReferenceGrants must stay
// authoritative for cross-namespace refs without syncing to the host.
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

//go:embed vcluster-gatewayapi-grants-disabled.yaml
var gatewayAPIGrantsDisabledVClusterYAML string

const gatewayAPIGrantsDisabledVClusterName = "gatewayapi-grants-disabled-vcluster"

func init() { suiteGatewayAPIGrantsDisabledVCluster() }

func suiteGatewayAPIGrantsDisabledVCluster() {
	// Ordered so all specs share one lazyvcluster bring-up; specs are independent.
	Describe("gatewayapi-grants-disabled-vcluster", labels.PR, labels.GatewayAPI, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx,
					gatewayAPIGrantsDisabledVClusterName,
					gatewayAPIGrantsDisabledVClusterYAML,
					lazyvcluster.WithPreSetup(setup.GatewayAPIPreSetup()),
				)
			})

			test_gatewayapi.GatewayAPIGrantsDisabledSpec()
		},
	)
}
