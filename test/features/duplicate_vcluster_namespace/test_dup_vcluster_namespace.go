package testfeatures

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
	pollingInterval     = time.Second * 2
	pollingDurationLong = time.Minute * 2
	filePath            = "commonValues.yaml"
	chartPath           = "../../../chart"
)

var _ = ginkgo.Describe("Test vCluster features", func() {
	f := framework.DefaultFramework
	ginkgo.BeforeEach(func() {
		ginkgo.By("Disconnect from any vCluster")
		common.DisconnectFromVCluster(f)
	})

	ginkgo.It("should verify no two virtual clusters can be deployed in same namespace ", func() {
		vClusterName := "t-vc-" + random.String(6)

		ginkgo.By("Deploy vCluster in default namespace")
		noDuplicateVirtualClusterInSameNamespace(vClusterName, f)

	})
	ginkgo.It("should verify two virtual clusters can be deployed in same namespace using the flag ", func() {
		vClusterName := "t-vc-" + random.String(6)

		ginkgo.By("Deploy second vCluster in same namespace")
		deploySecondVClusterInSameNamespace(vClusterName, f)

		ginkgo.By("Disconnect from any vCluster")
		common.DisconnectFromVCluster(f)

		ginkgo.By("Delete vCluster")
		common.DeleteVCluster(vClusterName, f)

	})
})
