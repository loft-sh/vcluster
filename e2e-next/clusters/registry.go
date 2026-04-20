// Package clusters defines shared cluster infrastructure for e2e tests.
//
// HostCluster (the kind host) is the only genuinely shared dependency and
// stays in the framework's dependency mechanism here. Per-test vCluster
// instances live next to the suite that uses them and are provisioned
// lazily via setup/lazyvcluster.LazyVCluster in a BeforeAll.
//
// DefaultVClusterOptions is the shared Helm/provider options bag the
// lazyvcluster package consumes when invoking cluster.Create.
package clusters

import (
	"os"
	"path/filepath"

	"github.com/loft-sh/e2e-framework/pkg/provider/kind"
	providervcluster "github.com/loft-sh/e2e-framework/pkg/provider/vcluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"sigs.k8s.io/e2e-framework/support"
)

// HostCluster is the kind host cluster used by every e2e suite. Stays in
// the framework dependency mechanism (eager provisioning in
// SynchronizedBeforeSuite) because every per-test vCluster runs on top of
// it.
var HostCluster = cluster.Define(
	cluster.WithName(constants.GetHostClusterName()),
	cluster.WithProvider(kind.NewProvider()),
	cluster.WithConfigFile("e2e-kind.config.yaml"),
)

// DefaultVClusterOptions are shared Helm/provider options applied to every
// per-test vCluster by lazyvcluster.LazyVCluster.
var DefaultVClusterOptions = []support.ClusterOpts{
	providervcluster.WithPath(filepath.Join(os.Getenv("GOBIN"), "vcluster")),
	providervcluster.WithLocalChartDir("../chart"),
	providervcluster.WithUpgrade(true),
	providervcluster.WithBackgroundProxyImage(constants.GetVClusterImage()),
}
