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
