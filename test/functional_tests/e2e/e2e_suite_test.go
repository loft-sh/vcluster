package e2e

import (
	"context"
	"testing"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	// Enable cloud provider auth
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	// Register tests
	_ "github.com/loft-sh/vcluster/test/functional_tests/e2e/coredns"
	_ "github.com/loft-sh/vcluster/test/functional_tests/e2e/k8sdefaultendpoint"
	_ "github.com/loft-sh/vcluster/test/functional_tests/e2e/manifests"
	_ "github.com/loft-sh/vcluster/test/functional_tests/e2e/node"
	_ "github.com/loft-sh/vcluster/test/functional_tests/e2e/servicesync"
	_ "github.com/loft-sh/vcluster/test/functional_tests/e2e/syncer/networkpolicies"
	_ "github.com/loft-sh/vcluster/test/functional_tests/e2e/syncer/pods"
	_ "github.com/loft-sh/vcluster/test/functional_tests/e2e/syncer/pvc"
	_ "github.com/loft-sh/vcluster/test/functional_tests/e2e/syncer/services"
	_ "github.com/loft-sh/vcluster/test/functional_tests/e2e/webhook"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	// API extensions are not in the above scheme set,
	// and must thus be added separately.
	_ = apiextensionsv1beta1.AddToScheme(scheme)
	_ = apiextensionsv1.AddToScheme(scheme)
	_ = apiregistrationv1.AddToScheme(scheme)
}

// TestRunE2ETests checks configuration parameters (specified through flags) and then runs
// E2E tests using the Ginkgo runner.
// If a "report directory" is specified, one or more JUnit test reports will be
// generated in this directory, and cluster logs will also be saved.
// This function is called on each Ginkgo node in parallel mode.
func TestRunE2ETests(t *testing.T) {
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

	ginkgo.RunSpecs(t, "Vcluster e2e suite")
}
