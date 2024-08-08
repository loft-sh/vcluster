package scheduler

import (
	"context"
	"reflect"
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = ginkgo.Describe("Scheduler sync", func() {
	f := framework.DefaultFramework
	ginkgo.It("Use taints and toleration to assign virtual node to pod", func() {
		ginkgo.By("Add taints to virtual nodes only")
		virtualNodes, err := f.VClusterClient.CoreV1().Nodes().List(f.Context, metav1.ListOptions{})
		framework.ExpectNoError(err)

		for _, vnode := range virtualNodes.Items {
			origNode := vnode.DeepCopy()
			vnode.Spec.Taints = append(vnode.Spec.Taints, corev1.Taint{
				Key:    "key1",
				Value:  "value1",
				Effect: corev1.TaintEffectNoSchedule,
			})

			patch := client.MergeFrom(origNode)
			patchBytes, err := patch.Data(&vnode)
			framework.ExpectNoError(err)

			_, err = f.VClusterClient.CoreV1().Nodes().Patch(f.Context, vnode.Name, patch.Type(), patchBytes, metav1.PatchOptions{})
			framework.ExpectNoError(err)
		}

		hostNodes, err := f.HostClient.CoreV1().Nodes().List(f.Context, metav1.ListOptions{})
		framework.ExpectNoError(err)

		hostNodesTaints := make(map[string][]corev1.Taint)
		for _, hnode := range hostNodes.Items {
			hostNodesTaints[hnode.Name] = hnode.Spec.Taints
		}

		virtualNodes, err = f.VClusterClient.CoreV1().Nodes().List(f.Context, metav1.ListOptions{})
		framework.ExpectNoError(err)

		virtualNodesTaints := make(map[string][]corev1.Taint)
		for _, vnode := range virtualNodes.Items {
			virtualNodesTaints[vnode.Name] = vnode.Spec.Taints
		}
		framework.ExpectEqual(false, reflect.DeepEqual(hostNodesTaints, virtualNodesTaints))

		ginkgo.By("Pod with matching toleration should be scheduled")
		nsName := "default"
		podName := "nginx"
		pod := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: podName,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: "nginx",
					},
				},
				Tolerations: []corev1.Toleration{
					{
						Key:      "key1",
						Operator: corev1.TolerationOpEqual,
						Value:    "value1",
						Effect:   corev1.TaintEffectNoSchedule,
					},
				},
			},
		}

		_, err = f.VClusterClient.CoreV1().Pods(nsName).Create(f.Context, pod, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		err = wait.PollUntilContextTimeout(f.Context, time.Second, time.Minute*2, false, func(ctx context.Context) (bool, error) {
			p, _ := f.VClusterClient.CoreV1().Pods(nsName).Get(ctx, podName, metav1.GetOptions{})
			if p.Status.Phase == corev1.PodRunning {
				return true, nil
			}
			return false, nil
		})
		framework.ExpectNoError(err)

		ginkgo.By("Pod without matching toleration should not be scheduled")
		pod1Name := "nginx1"
		pod1 := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: pod1Name,
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

		_, err = f.VClusterClient.CoreV1().Pods(nsName).Create(f.Context, pod1, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		err = wait.PollUntilContextTimeout(f.Context, time.Second, time.Minute*2, false, func(ctx context.Context) (bool, error) {
			p, _ := f.VClusterClient.CoreV1().Pods(nsName).Get(ctx, pod1Name, metav1.GetOptions{})
			if p.Status.Phase == corev1.PodRunning {
				return true, nil
			}
			return false, nil
		})
		framework.ExpectError(err)

		ginkgo.By("remove taints from virtual node and delete namespace from vcluster")
		vNodes, err := f.VClusterClient.CoreV1().Nodes().List(f.Context, metav1.ListOptions{})
		framework.ExpectNoError(err)

		for _, vnode := range vNodes.Items {
			vnode.Spec.Taints = vnode.Spec.Taints[:len(vnode.Spec.Taints)-1]
			_, err = f.VClusterClient.CoreV1().Nodes().Update(f.Context, &vnode, metav1.UpdateOptions{})
			framework.ExpectNoError(err)
		}

		virtualNodes, err = f.VClusterClient.CoreV1().Nodes().List(f.Context, metav1.ListOptions{})
		framework.ExpectNoError(err)

		virtualNodesTaints = make(map[string][]corev1.Taint)
		for _, vnode := range virtualNodes.Items {
			virtualNodesTaints[vnode.Name] = vnode.Spec.Taints
		}
		framework.ExpectEqual(true, reflect.DeepEqual(hostNodesTaints, virtualNodesTaints))

		ginkgo.By("delete pods from vcluster")
		err = f.VClusterClient.CoreV1().Pods(nsName).Delete(f.Context, podName, metav1.DeleteOptions{})
		framework.ExpectNoError(err)

		err = f.VClusterClient.CoreV1().Pods(nsName).Delete(f.Context, pod1Name, metav1.DeleteOptions{})
		framework.ExpectNoError(err)
	})
})
