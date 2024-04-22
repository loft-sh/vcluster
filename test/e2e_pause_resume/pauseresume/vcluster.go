package pauseresume

import (
	"context"
	"os/exec"
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = ginkgo.Describe("Pause Resume VCluster", func() {
	f := framework.DefaultFramework
	ginkgo.It("run vcluster pause and vcluster resume", func() {
		ginkgo.By("Make sure vcluster pods are running")
		pods, err := f.HostClient.CoreV1().Pods(f.VclusterNamespace).List(f.Context, metav1.ListOptions{
			LabelSelector: "app=vcluster",
		})
		framework.ExpectNoError(err)
		framework.ExpectEqual(true, len(pods.Items) > 0)

		ginkgo.By("Pause vcluster")
		cmd := exec.Command("vcluster", "pause", f.VclusterName, "-n", f.VclusterNamespace)
		err = cmd.Run()
		framework.ExpectNoError(err)

		pods, err = f.HostClient.CoreV1().Pods(f.VclusterNamespace).List(f.Context, metav1.ListOptions{
			LabelSelector: "app=vcluster",
		})
		framework.ExpectNoError(err)
		framework.ExpectEqual(true, len(pods.Items) == 0)

		ginkgo.By("Resume vcluster")
		cmd = exec.Command("vcluster", "resume", f.VclusterName, "-n", f.VclusterNamespace)
		err = cmd.Run()
		framework.ExpectNoError(err)

		err = wait.PollUntilContextTimeout(f.Context, time.Second, time.Minute*2, false, func(ctx context.Context) (done bool, err error) {
			newPods, _ := f.HostClient.CoreV1().Pods(f.VclusterNamespace).List(ctx, metav1.ListOptions{
				LabelSelector: "app=vcluster",
			})
			p := len(newPods.Items)
			if p > 0 {
				// rp, running pod counter
				rp := 0
				for _, pod := range newPods.Items {
					if pod.Status.Phase == corev1.PodRunning {
						rp = rp + 1
					}
				}
				if rp == p {
					return true, nil
				}
			}
			return false, nil
		})
		framework.ExpectNoError(err)
	})
})
