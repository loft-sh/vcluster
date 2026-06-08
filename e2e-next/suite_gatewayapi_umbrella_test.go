// Suite: gatewayapi-umbrella-vcluster
// vCluster: umbrella toHost.gatewayApi.enabled: true coverage (TC-02a variant A).
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

//go:embed vcluster-gatewayapi-umbrella.yaml
var gatewayAPIUmbrellaVClusterYAML string

const gatewayAPIUmbrellaVClusterName = "gatewayapi-umbrella-vcluster"

func init() { suiteGatewayAPIUmbrellaVCluster() }

func suiteGatewayAPIUmbrellaVCluster() {
	Describe("gatewayapi-umbrella-vcluster", labels.PR, labels.GatewayAPI, labels.GatewayClasses, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx,
					gatewayAPIUmbrellaVClusterName,
					gatewayAPIUmbrellaVClusterYAML,
					lazyvcluster.WithPreSetup(setup.GatewayAPIPreSetup()),
				)
			})

			test_gatewayapi.GatewayAPIUmbrellaSpec()
		},
	)
}
