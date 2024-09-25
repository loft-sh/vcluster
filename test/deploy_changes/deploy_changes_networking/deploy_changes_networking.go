package deploynetworkingchanges

import (
	"github.com/loft-sh/vcluster/test/deploy_changes/common"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
)

var _ = ginkgo.Describe("Deploy networking changes to vCluster", func() {
	f := framework.DefaultFramework
	ginkgo.It("Deploys new changes to vCluster", func() {

		ginkgo.By("Checking for no Resource Quota availability")
		framework.ExpectError(checkNoResourceQuota(f))

		ginkgo.By("Check if no Limit Range is available")
		framework.ExpectError(checkNoLimitRange(f))

		ginkgo.By("Check if no network policy is available")
		framework.ExpectError(checkNoNetworkPolicy(f))

		ginkgo.By("Enabling isolation policies in YAML")
		enableIsolationPolicies()

		ginkgo.By("Replacing placeholders in YAML")
		common.ReplaceYAMLPlaceholders()

		ginkgo.By("Deploying changes to vCluster")
		common.DeployChangesToVClusterUsingCLI(f)

		ginkgo.By("Disconnecting from vCluster")
		common.DisconnectFromVCluster()

		ginkgo.By("Verifying cluster is active")
		common.VerifyClusterIsActive(f)

		ginkgo.By("Checking for Resource Quota availability")
		framework.ExpectNoError(checkResourceQuota(f))

		ginkgo.By("Checking for Limit Range availability")
		framework.ExpectNoError(checkLimitRange(f))

		ginkgo.By("Checking for Network Policy availability")
		framework.ExpectNoError(checkNetworkPolicy(f))
	})
})
