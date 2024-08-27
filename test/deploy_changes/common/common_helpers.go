package common

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/gomega"
)

const (
	pollingInterval     = time.Second * 2
	pollingDurationLong = time.Minute * 2
	filePath            = "commonValues.yaml"
	chartPath           = "../../../chart"
)

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
	output, err := disconnectCmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "is not a virtual cluster context") {
			fmt.Println("No virtual cluster context to disconnect from.")
		} else {
			framework.ExpectNoError(err, "Error disconnecting from vCluster")
		}
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

func DeleteVCluster(vClusterName string, f *framework.Framework) {
	ctx, cancel := context.WithTimeout(context.Background(), pollingDurationLong)
	defer cancel()

	deleteCmd := exec.CommandContext(ctx, "vcluster", "delete", vClusterName, "-n", f.VclusterNamespace)
	var stdout, stderr bytes.Buffer
	deleteCmd.Stdout = &stdout
	deleteCmd.Stderr = &stderr
	err := deleteCmd.Run()
	if err != nil {
		fmt.Println("stderr: ", stderr.String())
		framework.ExpectNoError(err, "Error executing vcluster delete command")
	}
	gomega.Expect(strings.Contains(stdout.String(), "Successfully deleted virtual cluster")).To(gomega.BeTrue())
}
