package deploynetworkingchanges

import (
	"context"
	"testing"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	// Enable cloud provider auth
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
}

func TestRunDeployChangesTests(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	err := framework.CreateFramework(context.Background(), scheme)
	if err != nil {
		log.GetInstance().Fatalf("Error setting up framework: %v", err)
	}

	var _ = ginkgo.AfterSuite(func() {
		err = framework.DefaultFramework.Cleanup()
		if err != nil {
			log.GetInstance().Warnf("Error executing testsuite cleanup: %v", err)
		}
	})

	ginkgo.RunSpecs(t, "Vcluster Deploy changes suite")
}
