package deploychanges

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	pollingInterval     = time.Second * 2
	pollingDurationLong = time.Minute * 2
	filePath            = "./deploy_changes/commonValues.yaml"
	chartPath           = "../chart"
)

var _ = ginkgo.Describe("Deploy changes to vCluster", func() {
	f := framework.DefaultFramework
	ginkgo.It("SDeploys new changes to vCluster", func() {

		ginkgo.By("Check if no Resource Quota is available")
		_, err := f.HostClient.CoreV1().ResourceQuotas(f.VclusterNamespace).Get(f.Context, "vc-vcluster", metav1.GetOptions{})
		framework.ExpectError(err)

		ginkgo.By("Check if no Limit Range is available")
		_, err = f.HostClient.CoreV1().LimitRanges(f.VclusterNamespace).Get(f.Context, "vc-vcluster", metav1.GetOptions{})
		framework.ExpectError(err)

		ginkgo.By("Check if no network policy is available")
		_, err = f.HostClient.NetworkingV1().NetworkPolicies(f.VclusterNamespace).Get(f.Context, "vc-work-vcluster", metav1.GetOptions{})
		framework.ExpectError(err)

		var isolationParameters = []string{
			".policies.resourceQuota.enabled = true",
			".policies.limitRange.enabled = true",
			".policies.networkPolicy.enabled = true",
		}

		ginkgo.By("edit yaml to enable isolated workloads")
		for _, expr := range isolationParameters {
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			cmdExec := exec.Command("yq", "e", "-i", expr, filePath)
			cmdExec.Stdout = stdout
			cmdExec.Stderr = stderr
			err = cmdExec.Run()
			log.Println("Upgrade command output: ", stdout.String())
			log.Println("Upgrade command error output: ", stderr.String())
			log.Println("err: ", err)
			framework.ExpectNoError(err)
		}

		replaceCmd := exec.Command("sed", "-i", "s|REPLACE_REPOSITORY_NAME|"+os.Getenv("REPOSITORY_NAME")+"|g", filePath)
		err = replaceCmd.Run()
		framework.ExpectNoError(err)
		replaceCmd = exec.Command("sed", "-i", "s|REPLACE_TAG_NAME|"+os.Getenv("TAG_NAME")+"|g", filePath)
		err = replaceCmd.Run()
		framework.ExpectNoError(err)

		ginkgo.By("Deploy Changes")
		gomega.Eventually(func() bool {
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			checkCmd := exec.Command("vcluster", "create", "--upgrade", f.VclusterName, "--namespace", f.VclusterNamespace, "--local-chart-dir", chartPath, "-f", filePath)
			checkCmd.Stdout = stdout
			checkCmd.Stderr = stderr
			err := checkCmd.Run()
			log.Println("Upgrade command output: ", stdout.String())
			log.Println("Upgrade command error output: ", stderr.String())
			log.Println("err: ", err)
			framework.ExpectNoError(err)
			return err == nil && strings.Contains(stdout.String(), "Switched active kube context to")
		}).WithPolling(pollingInterval).WithTimeout(pollingDurationLong).Should(gomega.BeTrue())

		ginkgo.By("Disconnect from vcluster")
		disconnectCmd := cmd.NewDisconnectCmd(&flags.GlobalFlags{})
		disconnectCmd.SetArgs([]string{})

		err = disconnectCmd.Execute()
		if err != nil && !strings.Contains(err.Error(), "not a virtual cluster context") {
			framework.ExpectNoError(err)
		}

		ginkgo.By("Verify cluster is active")
		gomega.Eventually(func() bool {
			checkCmd := exec.Command("vcluster", "list")
			output, err := checkCmd.CombinedOutput()
			framework.ExpectNoError(err)
			return err == nil && strings.Contains(string(output), f.VclusterName) && strings.Contains(string(output), "Running")
		}).WithPolling(pollingInterval).WithTimeout(pollingDurationLong).Should(gomega.BeTrue())

		ginkgo.By("Check if Resource Quota is available")
		_, err = f.HostClient.CoreV1().ResourceQuotas(f.VclusterNamespace).Get(f.Context, "vc-"+f.VclusterName, metav1.GetOptions{})
		framework.ExpectNoError(err)

		ginkgo.By("Check if Limit Range is available")
		_, err = f.HostClient.CoreV1().LimitRanges(f.VclusterNamespace).Get(f.Context, "vc-"+f.VclusterName, metav1.GetOptions{})
		framework.ExpectNoError(err)

		ginkgo.By("Check if network policy is available")
		_, err = f.HostClient.NetworkingV1().NetworkPolicies(f.VclusterNamespace).Get(f.Context, "vc-work-"+f.VclusterName, metav1.GetOptions{})
		framework.ExpectNoError(err)
	})
})
