package rootless

import (
	"strings"

	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Rootless mode", func() {
	f := framework.DefaultFramework
	ginkgo.It("run vcluster in rootless mode", func() {
		pods, err := f.HostClient.CoreV1().Pods(f.VclusterNamespace).List(f.Context, metav1.ListOptions{
			LabelSelector: "app=vcluster",
		})
		framework.ExpectNoError(err)
		vclusterPod := pods.Items[0].Name
		cmd := []string{
			"/bin/sh",
			"-c",
			"id -u",
		}
		stdout, stderr, err := podhelper.ExecBuffered(
			f.Context,
			f.HostConfig,
			f.VclusterNamespace,
			vclusterPod,
			"syncer",
			cmd,
			nil,
		)
		framework.ExpectNoError(err)
		framework.ExpectEqual(0, len(stderr))
		framework.ExpectEqual("12345", strings.TrimSuffix(string(stdout), "\n"))
	})
})
