// Suite: ha-short-certs-vcluster
// Tests HA-coordinated cert rotation with lease-based coordination and
// workload rollouts for control-plane and deployed etcd propagation.
// vCluster: 2 replicas, 3m cert validity, 15s check interval.
// Run:      just run-e2e 'certs'
package e2e_next

import (
	"context"
	_ "embed"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/e2e-next/setup/lazyvcluster"
	"github.com/loft-sh/vcluster/e2e-next/test_security/certs"
	. "github.com/onsi/ginkgo/v2"
)

//go:embed vcluster-ha-short-certs.yaml
var haShortCertsVClusterYAML string

const haShortCertsVClusterName = "ha-short-certs-vcluster"

func init() {
	suiteHAShortCertsVCluster()
}

func suiteHAShortCertsVCluster() {
	Describe("ha-short-certs-vcluster", labels.Certs, Ordered,
		cluster.Use(clusters.HostCluster),
		func() {
			BeforeAll(func(ctx context.Context) context.Context {
				return lazyvcluster.LazyVCluster(ctx, haShortCertsVClusterName, haShortCertsVClusterYAML)
			})

			certs.HACertRotationSpec()
		},
	)
}
