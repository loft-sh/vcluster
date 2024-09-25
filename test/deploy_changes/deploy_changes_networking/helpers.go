package deploynetworkingchanges

import (
	"os/exec"
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	pollingInterval     = time.Second * 2
	pollingDurationLong = time.Minute * 2
	filePath            = "commonValues.yaml"
	chartPath           = "../../../chart"
)

func checkNoResourceQuota(f *framework.Framework) error {
	_, err := f.HostClient.CoreV1().ResourceQuotas(f.VclusterNamespace).Get(f.Context, "vc-vcluster", metav1.GetOptions{})
	return err
}

func checkNoLimitRange(f *framework.Framework) error {
	_, err := f.HostClient.CoreV1().LimitRanges(f.VclusterNamespace).Get(f.Context, "vc-vcluster", metav1.GetOptions{})
	return err
}

func checkNoNetworkPolicy(f *framework.Framework) error {
	_, err := f.HostClient.NetworkingV1().NetworkPolicies(f.VclusterNamespace).Get(f.Context, "vc-work-vcluster", metav1.GetOptions{})
	return err
}

func enableIsolationPolicies() {
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

func checkResourceQuota(f *framework.Framework) error {
	_, err := f.HostClient.CoreV1().ResourceQuotas(f.VclusterNamespace).Get(f.Context, "vc-"+f.VclusterName, metav1.GetOptions{})
	return err
}

func checkLimitRange(f *framework.Framework) error {
	_, err := f.HostClient.CoreV1().LimitRanges(f.VclusterNamespace).Get(f.Context, "vc-"+f.VclusterName, metav1.GetOptions{})
	return err
}

func checkNetworkPolicy(f *framework.Framework) error {
	_, err := f.HostClient.NetworkingV1().NetworkPolicies(f.VclusterNamespace).Get(f.Context, "vc-work-"+f.VclusterName, metav1.GetOptions{})
	return err
}
