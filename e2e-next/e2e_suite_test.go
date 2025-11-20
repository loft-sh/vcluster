package e2e_next

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/loft-sh/e2e-framework/pkg/e2e"
	cluster "github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/peterbourgon/ff/v3"

	"github.com/loft-sh/vcluster/e2e-next/constants"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/e2e-framework/support/kind"

	// Import tests
	_ "github.com/loft-sh/vcluster/e2e-next/test_core/sync"
	_ "github.com/loft-sh/vcluster/e2e-next/test_deploy"
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
	flag.StringVar(&vclusterImage, "vcluster-image", constants.GetVclusterImage(), "vCluster image to test")
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
	constants.SetVclusterImage(vclusterImage)

	e2e.SetSetupOnly(setupOnly)
	e2e.SetTeardown(!setupOnly && teardown)
	e2e.SetTeardownOnly(teardownOnly)
}

// This must be called before any ginkgo DSL evaluation
var _ = AddTreeConstructionNodeArgsTransformer(e2e.ContextualNodeTransformer)

func TestMain(m *testing.M) {
	handleFlags()
	os.Exit(m.Run())
}

func TestRunE2ETests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "vCluster E2E Suite", AroundNode(e2e.ContextualAroundNode))
}

var _ = BeforeSuite(func(ctx context.Context) context.Context {
	var err error

	// Disable Ginkgo's truncating of long lines'
	format.MaxLength = 0

	By("Creating kind cluster " + clusterName)
	clusterOptions := []cluster.Options{
		cluster.WithName(constants.GetHostClusterName()),
		cluster.WithProvider(kind.NewProvider()),
	}
	if constants.GetHostClusterName() == "kind-cluster" {
		clusterOptions = append(clusterOptions, cluster.WithConfigFile("e2e-kind.config.yaml"))
	}

	ctx, err = cluster.Create(clusterOptions...)(ctx)
	Expect(err).NotTo(HaveOccurred())

	By("Setting up controller runtime client for " + clusterName)
	ctx, err = cluster.SetupControllerRuntimeClient(cluster.WithCluster(clusterName))(ctx)
	Expect(err).NotTo(HaveOccurred())

	ctx, err = cluster.SetupKubeClient(clusterName)(ctx)
	Expect(err).NotTo(HaveOccurred())

	By("Setting current cluster to " + clusterName)
	ctx, err = cluster.UseCluster(clusterName)(ctx)
	Expect(err).NotTo(HaveOccurred())

	By("Loading image to kind cluster...")
	ctx, err = cluster.LoadImage(clusterName, vclusterImage)(ctx)
	Expect(err).NotTo(HaveOccurred())

	return ctx
})

var _ = AfterSuite(func(ctx context.Context) {
	_, err := cluster.Destroy(constants.GetHostClusterName())(ctx)
	Expect(err).NotTo(HaveOccurred())
})
