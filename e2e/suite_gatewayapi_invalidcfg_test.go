package e2e

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e/clusters"
	"github.com/loft-sh/vcluster/e2e/labels"
	"github.com/loft-sh/vcluster/e2e/setup"
	"github.com/loft-sh/vcluster/e2e/test_gatewayapi"
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
