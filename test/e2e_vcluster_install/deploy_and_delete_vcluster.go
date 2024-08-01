package e2evclusterinstall

import (
	"bytes"
	"os/exec"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform/random"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	pollingInterval     = time.Second * 2
	pollingDurationLong = time.Minute * 2
	vclusterRepo        = "https://charts.loft.sh"
	filePath            = "../commonValues.yaml"
)

var _ = ginkgo.Describe("Deploy and Delete vCluster", func() {
	ginkgo.BeforeEach(func() {
		disconnectCmd := cmd.NewDisconnectCmd(&flags.GlobalFlags{})
		disconnectCmd.SetArgs([]string{})

		err := disconnectCmd.Execute()
		if err != nil && !strings.Contains(err.Error(), "not a virtual cluster context") {
			framework.ExpectNoError(err)
		}
	})
	ginkgo.It("should deploy a vCluster using kubectl and delete it using kubectl", func() {
		var out bytes.Buffer
		vClusterName := "t-cluster-" + random.String(6)
		vClusterNamespace := "t-ns-" + random.String(6)

		createNamespaceCmd := exec.Command("kubectl", "create", "namespace", vClusterNamespace)
		err := createNamespaceCmd.Run()
		framework.ExpectNoError(err)

		helmCmd := exec.Command("helm", "template", vClusterName, "vcluster", "--repo", vclusterRepo, "-n", vClusterNamespace, "-f", filePath)
		helmCmd.Stdout = &out
		err = helmCmd.Run()
		framework.ExpectNoError(err)

		kubectlCmd := exec.Command("kubectl", "apply", "-f", "-")
		kubectlCmd.Stdin = &out
		err = kubectlCmd.Run()
		framework.ExpectNoError(err)

		gomega.Eventually(func() bool {

			checkCmd := exec.Command("vcluster", "list")
			output, err := checkCmd.CombinedOutput()
			framework.ExpectNoError(err)
			return err == nil && strings.Contains(string(output), vClusterName) && strings.Contains(string(output), "Running")
		}, pollingDurationLong, pollingInterval).Should(gomega.BeTrue(), "Virtual cluster failed to come up Active")

		deleteCmd := exec.Command("kubectl", "delete", "namespace", vClusterNamespace)
		err = deleteCmd.Run()
		framework.ExpectNoError(err)

		gomega.Eventually(func() bool {
			listCmd := cmd.NewListCmd(&flags.GlobalFlags{
				Namespace: vClusterNamespace,
			})
			listCmd.SetOut(&out)
			listCmd.SetErr(&out)
			err := listCmd.Execute()

			framework.ExpectNoError(err)
			return !strings.Contains(string(out.String()), vClusterName)
		}, pollingDurationLong, pollingInterval).Should(gomega.BeTrue(), "Virtual cluster failed to be deleted")

	})
})
