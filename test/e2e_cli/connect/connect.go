package connect

import (
	"os"
	"os/exec"
	"strings"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
)

var _ = ginkgo.Describe("Connect to vCluster", func() {
	f := framework.DefaultFramework
	ginkgo.BeforeEach(func() {
		disconnectCmd := cmd.NewDisconnectCmd(&flags.GlobalFlags{})
		disconnectCmd.SetArgs([]string{})

		err := disconnectCmd.Execute()
		if err != nil && !strings.Contains(err.Error(), "not a virtual cluster context") {
			framework.ExpectNoError(err)
		}
	})

	ginkgo.It("should connect to an OSS vcluster", func() {
		kcfgFile, err := os.CreateTemp("", "kubeconfig")
		framework.ExpectNoError(err)

		connectCmd := cmd.NewConnectCmd(&flags.GlobalFlags{})

		err = connectCmd.Flags().Set("kube-config", kcfgFile.Name())
		framework.ExpectNoError(err)

		connectCmd.SetArgs([]string{f.VclusterName})

		err = connectCmd.Execute()
		framework.ExpectNoError(err)
	})

	ginkgo.It("should print connect config to a file and use it to connect to an OSS vcluster", func() {
		kcfgFile, err := os.CreateTemp("", "kubeconfig")
		framework.ExpectNoError(err)
		// vcluster CLI has to be in $PATH
		connectCmd := exec.Command("vcluster", "connect", "-n", f.VclusterNamespace, "--print", f.VclusterName)
		kubeConfigBytes, err := connectCmd.Output()
		framework.ExpectNoError(err)
		_, err = kcfgFile.Write(kubeConfigBytes)
		framework.ExpectNoError(err)
		kubectlCmd := exec.Command("kubectl", "--kubeconfig", kcfgFile.Name(), "get", "pods", "-A")
		err = kubectlCmd.Run()
		framework.ExpectNoError(err)
	})

	ginkgo.It("should fail to print vcluster connect config to a file", func() {
		notExistingVClusterName := "INVALID"
		// vcluster CLI has to be in $PATH
		connectCmd := exec.Command("vcluster", "connect", "-n", notExistingVClusterName, "--print", notExistingVClusterName)
		err := connectCmd.Run()
		framework.ExpectError(err)
	})

	ginkgo.It("should connect to an OSS vcluster and execute a command", func() {
		connectCmd := cmd.NewConnectCmd(&flags.GlobalFlags{})
		connectCmd.SetArgs([]string{f.VclusterName, "--", "kubectl", "get", "ns"})

		err := connectCmd.Execute()
		framework.ExpectNoError(err)
	})
})
