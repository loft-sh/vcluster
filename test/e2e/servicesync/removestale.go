package servicesync

import (
	"time"

	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("previously replicated services removed from replication config", func() {
	var f *framework.Framework
	var vClusterNamespace string

	ginkgo.BeforeEach(func() {
		f = framework.DefaultFramework
		vClusterNamespace = f.VClusterNamespace
		if f.MultiNamespaceMode {
			vClusterNamespace = translate.NewMultiNamespaceTranslator(f.VClusterNamespace).HostNamespace(nil, "default")
		}
	})

	ginkgo.It("doesn't find virtual service that is removed from replication config", func() {
		// A test service 'test-replicated-service-cleanup' has been created in the vcluster that
		// is used for e2e tests, you can find it in 'test/e2e/values.yaml' in this repo. This
		// service mimics a previously replicated service, which is achieved by setting label
		// 'vcluster.loft.sh/controlled-by: vcluster'.
		// Since this service is not present in the .networking.replicateServices.fromHost config
		// it should be deleted when vcluster starts.

		gomega.Eventually(func() bool {
			_, err := f.VClusterClient.CoreV1().Services(vClusterNamespace).Get(f.Context, "test-replicated-service-cleanup", metav1.GetOptions{})
			return kerrors.IsNotFound(err)
		}).WithPolling(5 * time.Second).
			WithTimeout(framework.PollTimeout).
			Should(gomega.BeTrue())
	})
})
