package e2etargetnamespace

import (
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = ginkgo.Describe("Target Namespace", func() {
	f := framework.DefaultFramework
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

		_, err := f.VclusterClient.CoreV1().Pods("default").Create(f.Context, pod, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		err = wait.Poll(time.Second, time.Minute*2, func() (bool, error) {
			p, _ := f.VclusterClient.CoreV1().Pods("default").Get(f.Context, "nginx", metav1.GetOptions{})
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
		framework.ExpectEqual(true, len(p.Items) > 1)

		ginkgo.By("Check if OwnerReferences is set to pod")
		framework.ExpectEqual(*p.Items[0].OwnerReferences[0].Controller, true)

		ginkgo.By("Check if networkpolicy is created to target namespace")
		_, err = f.HostClient.NetworkingV1().NetworkPolicies("vcluster-workload").Get(f.Context, "vcluster-workloads", metav1.GetOptions{})
		framework.ExpectNoError(err)

		ginkgo.By("Check if resourcequota is created to target namespace")
		_, err = f.HostClient.CoreV1().ResourceQuotas("vcluster-workload").Get(f.Context, "vcluster-quota", metav1.GetOptions{})
		framework.ExpectNoError(err)

		ginkgo.By("Check if limitrange is created to target namespace")
		_, err = f.HostClient.CoreV1().LimitRanges("vcluster-workload").Get(f.Context, "vcluster-limit-range", metav1.GetOptions{})
		framework.ExpectNoError(err)
	})
})
