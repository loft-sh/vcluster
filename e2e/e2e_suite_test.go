package e2e

//
//import (
//	"math/rand"
//	"testing"
//	"time"
//
//	"github.com/onsi/ginkgo"
//	"github.com/onsi/gomega"
//	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
//	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
//	"k8s.io/apimachinery/pkg/runtime"
//	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
//	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
//	"k8s.io/klog"
//	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
//
//	// "go.uber.org/zap/zapcore"
//	// zappkg "go.uber.org/zap"
//
//	// Make sure the rest functions are registered
//	_ "github.com/loft-sh/loft/pkg/apiserver/registry/management"
//
//	// +kubebuilder:scaffold:imports
//
//	// Make sure dep tools picks up these dependencies
//	_ "github.com/go-openapi/loads"
//	_ "k8s.io/apimachinery/pkg/apis/meta/v1"
//	_ "k8s.io/client-go/plugin/pkg/client/auth" // Enable cloud provider auth
//
//	// Register tests
//)
//
//var (
//	scheme = runtime.NewScheme()
//)
//
//func init() {
//	_ = clientgoscheme.AddToScheme(scheme)
//	// API extensions are not in the above scheme set,
//	// and must thus be added separately.
//	_ = apiextensionsv1beta1.AddToScheme(scheme)
//	_ = apiextensionsv1.AddToScheme(scheme)
//	_ = apiregistrationv1.AddToScheme(scheme)
//}
//
//// TestRunE2ETests checks configuration parameters (specified through flags) and then runs
//// E2E tests using the Ginkgo runner.
//// If a "report directory" is specified, one or more JUnit test reports will be
//// generated in this directory, and cluster logs will also be saved.
//// This function is called on each Ginkgo node in parallel mode.
//func TestRunE2ETests(t *testing.T) {
//	rand.Seed(time.Now().UTC().UnixNano())
//	gomega.RegisterFailHandler(ginkgo.Fail)
//	err := setupFramework(scheme)
//	if err != nil {
//		klog.Fatalf("Error setting up framework: %v", err)
//	}
//
//	ginkgo.RunSpecs(t, "DevSpace e2e suite")
//}
