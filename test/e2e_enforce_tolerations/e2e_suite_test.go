package e2eenforcetolerations

import (
	"context"
	"testing"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	// Register tests
	_ "github.com/loft-sh/vcluster/test/e2e_enforce_tolerations/pods"
)

// TestRunE2EEnforceTolerationTests verifies that vcluster's enforceTolerations configuration
// (sync.toHost.pods.enforceTolerations) is applied to physical pods both at creation time
// and when virtual pod tolerations are updated.
func TestRunE2EEnforceTolerationTests(t *testing.T) {
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

	ginkgo.RunSpecs(t, "VCluster enforce tolerations e2e suite")
}
