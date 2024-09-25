package deploynetworkingchanges

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	pollingInterval     = time.Second * 2
	pollingDurationLong = time.Minute * 2
	filePath            = "commonValues.yaml"
	chartPath           = "../../../chart"
)

func CheckNoResourceQuota(f *framework.Framework) error {
	_, err := f.HostClient.CoreV1().ResourceQuotas(f.VclusterNamespace).Get(f.Context, "vc-vcluster", metav1.GetOptions{})
	return err
}

func CheckNoLimitRange(f *framework.Framework) error {
	_, err := f.HostClient.CoreV1().LimitRanges(f.VclusterNamespace).Get(f.Context, "vc-vcluster", metav1.GetOptions{})
	return err
}

func CheckNoNetworkPolicy(f *framework.Framework) error {
	_, err := f.HostClient.NetworkingV1().NetworkPolicies(f.VclusterNamespace).Get(f.Context, "vc-work-vcluster", metav1.GetOptions{})
	return err
}

func EnableIsolationPolicies() {
	isolationParameters := []string{
		".policies.resourceQuota.enabled = true",
		".policies.limitRange.enabled = true",
		".policies.networkPolicy.enabled = true",
	}

	for _, expr := range isolationParameters {
		cmdExec := exec.Command("yq", "e", "-i", expr, filePath)
		err := cmdExec.Run()
		framework.ExpectNoError(err, "Failure to edit file")
	}
}

func ReplaceYAMLPlaceholders() {
	replaceCmd := exec.Command("sed", "-i", "s|REPLACE_REPOSITORY_NAME|"+os.Getenv("REPOSITORY_NAME")+"|g", filePath)
	err := replaceCmd.Run()
	framework.ExpectNoError(err, "Failure to edit file")

	replaceCmd = exec.Command("sed", "-i", "s|REPLACE_TAG_NAME|"+os.Getenv("TAG_NAME")+"|g", filePath)
	err = replaceCmd.Run()
	framework.ExpectNoError(err, "Failure to edit file")
}

func DeployChangesToVClusterUsingCLI(f *framework.Framework) {
	gomega.Eventually(func() bool {
		stdout := &bytes.Buffer{}
		deployCmd := exec.Command("vcluster", "create", "--upgrade", f.VclusterName, "--namespace", f.VclusterNamespace, "--local-chart-dir", chartPath, "-f", filePath)
		deployCmd.Stdout = stdout
		err := deployCmd.Run()
		framework.ExpectNoError(err)
		return err == nil && strings.Contains(stdout.String(), "Switched active kube context to")
	}).WithPolling(pollingInterval).WithTimeout(pollingDurationLong).Should(gomega.BeTrue())
}

func DisconnectFromVCluster(f *framework.Framework) {
	disconnectCmd := exec.Command("vcluster", "disconnect")
	err := disconnectCmd.Run()
	if err != nil && !strings.Contains(err.Error(), "not a virtual cluster context") {
		framework.ExpectNoError(err, "Error disconnecting from vCluster")
	}
}

func VerifyClusterIsActive(f *framework.Framework) {
	gomega.Eventually(func() bool {
		checkCmd := exec.Command("vcluster", "list")
		output, err := checkCmd.CombinedOutput()
		framework.ExpectNoError(err)
		return err == nil && strings.Contains(string(output), f.VclusterName) && strings.Contains(string(output), "Running")
	}).WithPolling(pollingInterval).WithTimeout(pollingDurationLong).Should(gomega.BeTrue())
}

func CheckResourceQuota(f *framework.Framework) error {
	_, err := f.HostClient.CoreV1().ResourceQuotas(f.VclusterNamespace).Get(f.Context, "vc-"+f.VclusterName, metav1.GetOptions{})
	return err
}

func CheckLimitRange(f *framework.Framework) error {
	_, err := f.HostClient.CoreV1().LimitRanges(f.VclusterNamespace).Get(f.Context, "vc-"+f.VclusterName, metav1.GetOptions{})
	return err
}

func CheckNetworkPolicy(f *framework.Framework) error {
	_, err := f.HostClient.NetworkingV1().NetworkPolicies(f.VclusterNamespace).Get(f.Context, "vc-work-"+f.VclusterName, metav1.GetOptions{})
	return err
}
