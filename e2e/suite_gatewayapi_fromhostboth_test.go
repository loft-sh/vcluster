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

//go:embed vcluster-gatewayapi-fromhost-both.yaml
var gatewayAPIFromHostBothVClusterYAML string

const gatewayAPIFromHostBothVClusterName = "gatewayapi-fromhostboth-vcluster"

func init() { suiteGatewayAPIFromHostBothVCluster() }

func suiteGatewayAPIFromHostBothVCluster() {
	Describe("gatewayapi-fromhostboth-vcluster", labels.GatewayAPI, labels.GatewayClasses, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx,
					gatewayAPIFromHostBothVClusterName,
					gatewayAPIFromHostBothVClusterYAML,
					lazyvcluster.WithPreSetup(setup.GatewayAPIPreSetup()),
				)
			})

			test_gatewayapi.GatewayAPIFromHostBothSpec()
		},
	)
}
