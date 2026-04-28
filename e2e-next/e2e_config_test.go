package e2e_next

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/loft-sh/e2e-framework/pkg/e2e"
	"github.com/loft-sh/e2e-framework/pkg/setup"
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/peterbourgon/ff/v3"

	"github.com/loft-sh/vcluster/e2e-next/constants"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	// Initialize framework
	_ "github.com/loft-sh/vcluster/e2e-next/init"
)

var (
	vclusterImage string
	clusterName   string
	setupOnly     bool
	teardown      bool
	teardownOnly  bool
)

// Register your flags in an init function.  This ensures they are registered _before_ `go test` calls flag.Parse().
func handleFlags() {
	flag.StringVar(&vclusterImage, "vcluster-image", constants.GetVClusterImage(), "vCluster image to test")
	flag.StringVar(&clusterName, "cluster-name", constants.GetHostClusterName(), "The kind cluster to run test against. Optional.")
	flag.BoolVar(&setupOnly, "setup-only", false, "Skip tests and setup the environment")
	flag.BoolVar(&teardown, "teardown", true, "Disables [e2e.AfterSuite] [e2e.AfterAll] to leave environment in place.")
	flag.BoolVar(&teardownOnly, "teardown-only", false, "Skip tests and tear down the environment")

	err := ff.Parse(flag.CommandLine, os.Args[1:],
		ff.WithEnvVars(),
	)
	if err != nil {
		panic(err)
	}

	constants.SetHostClusterName(clusterName)
	constants.SetVClusterImage(vclusterImage)

	e2e.SetSetupOnly(setupOnly)
	e2e.SetTeardown(!setupOnly && teardown)
	e2e.SetTeardownOnly(teardownOnly)
}

func TestMain(m *testing.M) {
	handleFlags()
	os.Exit(m.Run())
}

func TestRunE2ETests(t *testing.T) {
	// Disable Ginkgo's truncating of long lines'
	format.MaxLength = 0
	config, _ := GinkgoConfiguration()

	RegisterFailHandler(Fail)
	RunSpecs(
		t,
		"vCluster E2E Suite",
		AroundNode(suite.PreviewSpecsAroundNode(config)),
		AroundNode(e2e.ContextualAroundNode),
	)
}

var _ = SynchronizedBeforeSuite(
	func(ctx context.Context) (context.Context, []byte) {
		var err error

		ctx, err = setup.All(
			clusters.HostCluster.Setup,
			func(ctx context.Context) (context.Context, error) {
				if cluster.From(ctx, clusterName) == nil {
					return ctx, nil
				}
				var err error
				By("Loading image to kind cluster...", func() {
					ctx, err = cluster.LoadImage(clusterName, vclusterImage)(ctx)
					Expect(err).To(Succeed())
				})
				return ctx, err
			},
			// Per-test vClusters are now provisioned lazily in each suite's
			// BeforeAll via setup/lazyvcluster.LazyVCluster - no upfront
			// provisioning step needed here.
		)(ctx)
		Expect(err).To(Succeed())

		data, err := cluster.ExportAll(ctx)
		Expect(err).To(Succeed())

		return ctx, data
	},
	func(ctx context.Context, data []byte) context.Context {
		var err error

		ctx, err = cluster.ImportAll(ctx, data)
		Expect(err).To(Succeed())

		return ctx
	},
)

var _ = SynchronizedAfterSuite(
	func(ctx context.Context) {
	},
	func(ctx context.Context) {
		_, err := setup.All(
			clusters.HostCluster.Teardown,
		)(ctx)
		Expect(err).To(Succeed())
	},
)
