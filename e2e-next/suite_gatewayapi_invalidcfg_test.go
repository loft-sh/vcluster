// Suite: gatewayapi-invalidcfg
// Pure config-validation specs (TC-38/39/40/41) — no per-suite vCluster.
// Each spec shells out to `vcluster create` with intentionally broken YAML
// and asserts the CLI rejects it at deploy time. Will fail until
// ENGNODE-554 / ENGNODE-555 / ENGNODE-556 land.
// Run:      just run-e2e 'pr && gatewayapi'
package e2e_next

import (
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/e2e-next/test_gatewayapi"
	. "github.com/onsi/ginkgo/v2"
)

func init() { suiteGatewayAPIInvalidConfig() }

func suiteGatewayAPIInvalidConfig() {
	Describe("gatewayapi-invalidcfg", labels.GatewayAPI, labels.GatewayClasses,
		cluster.Use(clusters.HostCluster),
		func() {
			test_gatewayapi.GatewayAPIInvalidConfigSpec()
		},
	)
}
