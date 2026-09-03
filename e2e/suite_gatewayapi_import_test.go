// Suite: gatewayapi-import-vcluster
// vCluster: fromHost imported Gateway coverage (mirroring, allowedRoutes policy,
// hostname allowlist, deletion recovery, read-only behavior, status sanitization).
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

//go:embed vcluster-gatewayapi-import.yaml
var gatewayAPIImportVClusterYAML string

const gatewayAPIImportVClusterName = "gatewayapi-import-vcluster"

func init() { suiteGatewayAPIImportVCluster() }

func suiteGatewayAPIImportVCluster() {
	// Ordered so all specs share one lazyvcluster bring-up; specs are independent.
	Describe("gatewayapi-import-vcluster", labels.PR, labels.GatewayAPI, labels.GatewayClasses, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx,
					gatewayAPIImportVClusterName,
					gatewayAPIImportVClusterYAML,
					lazyvcluster.WithPreSetup(setup.GatewayAPIPreSetup()),
				)
			})

			test_gatewayapi.GatewayAPIImportSpec()
		},
	)
}
