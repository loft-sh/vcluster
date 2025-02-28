package e2egeneric

import (
	"context"
	"testing"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	// Enable cloud provider aut
	// Enable cloud provider auth
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	// Register tests
	_ "github.com/loft-sh/vcluster/test/e2e_generic/clusterscope"
)

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

	ginkgo.RunSpecs(t, "Vcluster e2e suite")
}
