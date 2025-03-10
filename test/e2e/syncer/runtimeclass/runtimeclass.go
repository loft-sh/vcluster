package runtimeclass

import (
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	nodev1 "k8s.io/api/node/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Verify runtimeClass is synced from Host to vCluster", ginkgo.Ordered, func() {
	var (
		f                *framework.Framework
		runtimeClass     *nodev1.RuntimeClass
		runtimeClassName = "my-custom-runtime"
	)

	ginkgo.BeforeAll(func() {
		f = framework.DefaultFramework
		runtimeClass = &nodev1.RuntimeClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: runtimeClassName,
			},
			Handler: "custom-handler",
		}
	})
	ginkgo.AfterAll(func() {
		framework.ExpectNoError(f.HostClient.NodeV1().RuntimeClasses().Delete(f.Context, runtimeClassName, metav1.DeleteOptions{}))
	})

	ginkgo.It("Runtime Class is synced from Host to vCluster", func() {
		_, err := f.HostClient.NodeV1().RuntimeClasses().Create(f.Context, runtimeClass, metav1.CreateOptions{})
		framework.ExpectNoError(err)
		var runtimeClass1 *nodev1.RuntimeClass
		gomega.Eventually(func() bool {
			runtimeClass1, err = f.VClusterClient.NodeV1().RuntimeClasses().Get(f.Context, runtimeClassName, metav1.GetOptions{})
			return err == nil
		}).WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(gomega.BeTrue())

		gomega.Expect(runtimeClass1.Name).To(gomega.Equal(runtimeClassName))
	})

})
