package testfeatures

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/gomega"
)

func noDuplicateVirtualClusterInSameNamespace(vClusterName string, f *framework.Framework) {
	deployCmd := exec.Command("vcluster", "create", vClusterName, "--namespace", f.VclusterNamespace, "-f", filePath)
	output, err := deployCmd.CombinedOutput()
	framework.ExpectError(err)
	gomega.Expect(strings.Contains(string(output), "there is already a virtual cluster in namespace")).To(gomega.BeTrue())
}

func deploySecondVClusterInSameNamespace(vClusterName string, f *framework.Framework) {
	gomega.Eventually(func() bool {
		stdout := &bytes.Buffer{}
		deployCmd := exec.Command("vcluster", "create", vClusterName, "--namespace", f.VclusterNamespace, "-f", filePath, "--reuse-namespace")
		deployCmd.Stdout = stdout
		err := deployCmd.Run()
		framework.ExpectNoError(err)
		return err == nil && strings.Contains(stdout.String(), "Switched active kube context to")
	}).WithPolling(pollingInterval).WithTimeout(pollingDurationLong).Should(gomega.BeTrue())
}
