package e2e

import (
	"context"
	"testing"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	// Enable cloud provider aut
	// Enable cloud provider auth
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	// Register tests
	_ "github.com/loft-sh/vcluster/test/e2e/coredns"
	_ "github.com/loft-sh/vcluster/test/e2e/k8sdefaultendpoint"
	_ "github.com/loft-sh/vcluster/test/e2e/manifests"
	_ "github.com/loft-sh/vcluster/test/e2e/node"
	_ "github.com/loft-sh/vcluster/test/e2e/servicesync"
	_ "github.com/loft-sh/vcluster/test/e2e/snapshot"
	_ "github.com/loft-sh/vcluster/test/e2e/syncer/fromhost"
	_ "github.com/loft-sh/vcluster/test/e2e/syncer/networkpolicies"
	_ "github.com/loft-sh/vcluster/test/e2e/syncer/pods"
	_ "github.com/loft-sh/vcluster/test/e2e/syncer/pvc"
	_ "github.com/loft-sh/vcluster/test/e2e/syncer/runtimeclass"
	_ "github.com/loft-sh/vcluster/test/e2e/syncer/services"
	_ "github.com/loft-sh/vcluster/test/e2e/syncer/tohost"
	_ "github.com/loft-sh/vcluster/test/e2e/webhook"
)

// TestRunE2ETests checks configuration parameters (specified through flags) and then runs
// E2E tests using the Ginkgo runner.
// If a "report directory" is specified, one or more JUnit test reports will be
// generated in this directory, and cluster logs will also be saved.
// This function is called on each Ginkgo node in parallel mode.
func TestRunE2ETests(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	err := framework.CreateFramework(context.Background())
	if err != nil {
		log.GetInstance().Fatalf("Error setting up framework: %v", err)
	}

	var _ = ginkgo.AfterSuite(func() {
		err = framework.DefaultFramework.Cleanup()
		if err != nil {
			log.GetInstance().Warnf("Error executing testsuite cleanup: %v", err)
		}
	})

	ginkgo.RunSpecs(t, "VCluster e2e suite")
}
