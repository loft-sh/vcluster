package e2e_next

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/e2e-next/setup"
	"github.com/loft-sh/vcluster/e2e-next/test_gatewayapi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func init() { suiteGatewayAPIInvalidConfig() }

func suiteGatewayAPIInvalidConfig() {
	Describe("gatewayapi-invalidcfg", labels.GatewayAPI, labels.GatewayClasses, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) {
				Expect(setup.GatewayAPIPreSetup()(ctx)).To(Succeed())
			})

			test_gatewayapi.GatewayAPIInvalidConfigSpec()
		},
	)
}
