package targetnamespace

import (
	"context"
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = ginkgo.Describe("Target Namespace", func() {
	f := framework.DefaultFramework
	if f.MultiNamespaceMode {
		ginkgo.Skip("Target Namespace tests are not applicable in Multi-Namespace mode")
	}

	ginkgo.It("Create vcluster with target namespace", func() {
		ginkgo.By("Create workload in vcluster and verify if it's running in targeted namespace")
		pod := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "nginx",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: "nginx",
					},
				},
			},
		}

		_, err := f.VClusterClient.CoreV1().Pods("default").Create(f.Context, pod, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		err = wait.PollUntilContextTimeout(f.Context, time.Second, time.Minute*2, false, func(ctx context.Context) (bool, error) {
			p, _ := f.VClusterClient.CoreV1().Pods("default").Get(ctx, "nginx", metav1.GetOptions{})
			if p.Status.Phase == corev1.PodRunning {
				return true, nil
			}
			return false, nil
		})
		framework.ExpectNoError(err)

		p, err := f.HostClient.CoreV1().Pods("vcluster-workload").List(f.Context, metav1.ListOptions{
			LabelSelector: "vcluster.loft.sh/managed-by=" + f.VclusterName,
		})
		framework.ExpectNoError(err)
		framework.ExpectEqual(true, len(p.Items) > 0)
	})
})
