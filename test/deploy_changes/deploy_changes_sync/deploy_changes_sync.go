package deploysyncchanges

import (
	"time"

	"github.com/loft-sh/vcluster/pkg/platform/random"
	"github.com/loft-sh/vcluster/test/deploy_changes/common"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	scheme = runtime.NewScheme()
)

const (
	pollingInterval       = time.Second * 2
	pollingDurationLong   = time.Minute * 2
	vclusterRepo          = "https://charts.loft.sh"
	filePath              = "commonValues.yaml"
	chartPath             = "../../../chart"
	initialNsLabelKey     = "testing-ns-label"
	initialNsLabelValue   = "testing-ns-label-value"
	testingContainerName  = "nginx"
	testingContainerImage = "nginxinc/nginx-unprivileged"
)

var _ = ginkgo.Describe("Deploy sync changes to vCluster", func() {
	ginkgo.It("should deploy a vCluster using kubectl and verify sync changes ", func() {
		f := framework.DefaultFramework
		saName := "t-sa-" + random.String(6)
		podName := "t-pod-" + random.String(6)
		testNamespace := "t-ns-" + random.String(6)

		ginkgo.By("Create test namespace in vCluster")
		createVClusterTestNamespace(testNamespace, f)

		ginkgo.By("Create service account and pod in vCluster")
		createServiceAccount(testNamespace, saName, f)
		createPod(testNamespace, podName, f)

		ginkgo.By("Verifying pod, service account sync status")
		verifyPodSyncToHostStatus(testNamespace, podName, f, true)
		verifyServiceAccountSyncToHostStatus(testNamespace, saName, f, false)

		ginkgo.By("Updating sync settings for vCluster")
		syncSettings := []string{
			".sync.toHost.serviceAccounts.enabled=true",
		}
		updateVClusterSyncSettings(syncSettings)

		ginkgo.By("Replacing placeholders in YAML")
		common.ReplaceYAMLPlaceholders()

		ginkgo.By("Deploying changes to vCluster")
		common.DeployChangesToVClusterUsingCLI(f)

		ginkgo.By("Disconnecting from vCluster")
		common.DisconnectFromVCluster(f)

		ginkgo.By("Verifying cluster is active")
		common.VerifyClusterIsActive(f)

		ginkgo.By("Verifying service account sync to Host")
		verifyServiceAccountSyncToHostStatus(testNamespace, saName, f, true)

	})
})
