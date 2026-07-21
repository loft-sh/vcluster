// Suite: gatewayapi-vcluster
// vCluster: Gateway API sync coverage for GatewayClass visibility, Tenant
// Gateway authorization, and Gateway/HTTPRoute sync.
// Run:      just run-e2e 'pr && gatewayapi'
package e2e

import (
	"context"
	_ "embed"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e/clusters"
	"github.com/loft-sh/vcluster/e2e/labels"
	"github.com/loft-sh/vcluster/e2e/setup"
	"github.com/loft-sh/vcluster/e2e/setup/lazyvcluster"
	"github.com/loft-sh/vcluster/e2e/test_gatewayapi"
	. "github.com/onsi/ginkgo/v2"
)

//go:embed vcluster-gatewayapi.yaml
var gatewayAPIVClusterYAML string

const gatewayAPIVClusterName = "gatewayapi-vcluster"

func init() { suiteGatewayAPIVCluster() }

func suiteGatewayAPIVCluster() {
	// Ordered so all specs share one lazyvcluster bring-up; specs are independent.
	Describe("gatewayapi-vcluster", labels.PR, labels.GatewayAPI, labels.GatewayClasses, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx,
					gatewayAPIVClusterName,
					gatewayAPIVClusterYAML,
					lazyvcluster.WithPreSetup(setup.GatewayAPIPreSetup()),
				)
			})

			test_gatewayapi.GatewayAPISyncSpec()
			test_gatewayapi.GatewayAPIToHostSpec()
		},
	)
}
