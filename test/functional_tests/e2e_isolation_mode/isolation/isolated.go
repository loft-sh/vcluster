package isolation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = ginkgo.Describe("Isolated mode", func() {
	f := framework.DefaultFramework
	if f.MultiNamespaceMode {
		ginkgo.Skip("Isolated mode is not supported in Multi-Namespace mode")
	}

	ginkgo.It("Enforce isolated mode", func() {
		ginkgo.By("Check if isolated mode creates resourcequota")
		_, err := f.HostClient.CoreV1().ResourceQuotas(f.VclusterNamespace).Get(f.Context, "vc-vcluster", metav1.GetOptions{})
		framework.ExpectNoError(err)

		ginkgo.By("Check if isolated mode creates limitrange")
		_, err = f.HostClient.CoreV1().LimitRanges(f.VclusterNamespace).Get(f.Context, "vc-vcluster", metav1.GetOptions{})
		framework.ExpectNoError(err)

		ginkgo.By("Check if isolated mode creates networkpolicy")
		_, err = f.HostClient.NetworkingV1().NetworkPolicies(f.VclusterNamespace).Get(f.Context, "vc-work-vcluster", metav1.GetOptions{})
		framework.ExpectNoError(err)

		ginkgo.By("Check if isolated mode applies baseline PodSecurityStandards to namespaces in vcluster")
		ns, err := f.VClusterClient.CoreV1().Namespaces().Get(f.Context, "default", metav1.GetOptions{})
		framework.ExpectNoError(err)
		if ns.Labels["pod-security.kubernetes.io/enforce"] != "baseline" {
			framework.Failf("baseline PodSecurityStandards is not applied")
		}

		ginkgo.By("Check if isolated mode applies baseline PodSecurityStandards to new namespace in vcluster")
		nsName := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-new-namespace",
			},
		}
		_, err = f.VClusterClient.CoreV1().Namespaces().Create(f.Context, nsName, metav1.CreateOptions{})
		framework.ExpectNoError(err)
		err = wait.PollUntilContextTimeout(f.Context, time.Second, time.Minute, false, func(ctx context.Context) (done bool, err error) {
			ns, _ := f.VClusterClient.CoreV1().Namespaces().Get(ctx, nsName.Name, metav1.GetOptions{})
			if ns.Status.Phase == corev1.NamespaceActive {
				return true, nil
			}
			return false, nil
		})
		framework.ExpectNoError(err)
		if ns.Labels["pod-security.kubernetes.io/enforce"] != "baseline" {
			framework.Failf("baseline PodSecurityStandards is not applied for new namespace")
		}

		err = f.VClusterClient.CoreV1().Namespaces().Delete(f.Context, nsName.Name, metav1.DeleteOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.It("Fails to schedule pod violating resourcequota and limitrange", func() {
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
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU: resource.MustParse("2"),
							},
						},
					},
				},
			},
		}

		_, err := f.VClusterClient.CoreV1().Pods("default").Create(f.Context, pod, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		err = wait.PollUntilContextTimeout(f.Context, time.Second*2, time.Minute*1, false, func(ctx context.Context) (bool, error) {
			p, _ := f.VClusterClient.CoreV1().Pods("default").Get(ctx, "nginx", metav1.GetOptions{})
			if p.Status.Phase == corev1.PodRunning {
				return true, nil
			}

			e, _ := f.VClusterClient.CoreV1().Events("default").List(ctx, metav1.ListOptions{TypeMeta: p.TypeMeta})
			if len(e.Items) > 0 {
				if strings.Contains(e.Items[0].Message, `Invalid value: "2": must be less than or equal to cpu limit`) {
					return true, fmt.Errorf(`invalid value: "2": must be less than or equal to cpu limit`)
				}
			}
			return false, nil
		})
		framework.ExpectError(err)

		err = f.VClusterClient.CoreV1().Pods("default").Delete(f.Context, pod.Name, metav1.DeleteOptions{})
		framework.ExpectNoError(err)
	})
})
