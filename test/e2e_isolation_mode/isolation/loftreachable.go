package isolation

import (
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
)

var _ = ginkgo.Describe("test that we can reach pod with a app:loft label", func() {
	f := framework.DefaultFramework

	if f.MultiNamespaceMode {
		ginkgo.Skip("Isolated mode is not supported in Multi-Namespace mode")
	}

	cpod, err := f.CreateCurlPodHost(framework.DefaultFramework.VclusterNamespace, map[string]string{"release": framework.DefaultFramework.VclusterName})
	framework.ExpectNoError(err)

	ginkgo.It("should be able to reach a pod in another ns if it has a app:loft", func() {
		_, svc, err := f.CreateHostNginxPodAndService("test-ns", map[string]string{"app": "loft"})
		framework.ExpectNoError(err)

		framework.DefaultFramework.TestHostServiceIsEventuallyReachable(cpod, svc)
	})

	ginkgo.It("should not be able to reach a pod in another ns if it has a app:something", func() {
		_, svc, err := f.CreateHostNginxPodAndService("test-ns-2", map[string]string{"app": "something"})
		framework.ExpectNoError(err)
		framework.DefaultFramework.TestHostServiceIsEventuallyUnreachable(cpod, svc)
	})
})
