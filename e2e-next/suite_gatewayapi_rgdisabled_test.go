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

//go:embed vcluster-gatewayapi-rg-disabled.yaml
var gatewayAPIRGDisabledVClusterYAML string

const gatewayAPIRGDisabledVClusterName = "gatewayapi-rgdisabled-vcluster"

func init() { suiteGatewayAPIRGDisabledVCluster() }

func suiteGatewayAPIRGDisabledVCluster() {
	Describe("gatewayapi-rgdisabled-vcluster", labels.GatewayAPI, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx,
					gatewayAPIRGDisabledVClusterName,
					gatewayAPIRGDisabledVClusterYAML,
					lazyvcluster.WithPreSetup(setup.GatewayAPIPreSetup()),
				)
			})

			test_gatewayapi.GatewayAPIReferenceGrantDisabledSpec()
		},
	)
}
