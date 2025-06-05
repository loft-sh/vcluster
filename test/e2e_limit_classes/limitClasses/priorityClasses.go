package limitclasses

import (
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Test limitclass on fromHost", ginkgo.Ordered, func() {
	var (
		f                  *framework.Framework
		hpriorityClassName = "high-priority"
		lpriorityClassName = "low-priority"

		labelValue1 = "one"
		labelValue2 = "two"

		hpPodName = "hp-pod"
		lpPodName = "lp-pod"

		testNamespace = "default"
		hostNamespace = "vcluster"
	)

	ginkgo.BeforeAll(func() {
		f = framework.DefaultFramework
		// Create high priority priorityClass on host
		hpPriorityClass := &schedulingv1.PriorityClass{
			ObjectMeta: metav1.ObjectMeta{
				Name:   hpriorityClassName,
				Labels: map[string]string{"value": labelValue1},
			},
			Value:         1000000,
			GlobalDefault: false,
			Description:   "This priorityClass should be used for high-priority workloads.",
		}
		_, err := f.HostClient.SchedulingV1().PriorityClasses().Create(f.Context, hpPriorityClass, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		// Create low priority priorityClass on host
		lpPriorityClass := &schedulingv1.PriorityClass{
			ObjectMeta: metav1.ObjectMeta{
				Name:   lpriorityClassName,
				Labels: map[string]string{"value": labelValue2},
			},
			Value:         10000,
			GlobalDefault: false,
			Description:   "This priorityClass should be used for low-priority workloads.",
		}
		_, err = f.HostClient.SchedulingV1().PriorityClasses().Create(f.Context, lpPriorityClass, metav1.CreateOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.AfterAll(func() {
		_ = f.HostClient.SchedulingV1().PriorityClasses().Delete(f.Context, hpriorityClassName, metav1.DeleteOptions{})
		_ = f.HostClient.SchedulingV1().PriorityClasses().Delete(f.Context, lpriorityClassName, metav1.DeleteOptions{})
		_ = f.HostClient.CoreV1().Pods(testNamespace).Delete(f.Context, hpPodName, metav1.DeleteOptions{})
		_ = f.HostClient.CoreV1().Pods(testNamespace).Delete(f.Context, lpPodName, metav1.DeleteOptions{})
	})

	ginkgo.It("should only sync priorityClasses to virtual with allowed label", func() {
		gomega.Eventually(func() []string {
			ics, err := f.VClusterClient.SchedulingV1().PriorityClasses().List(f.Context, metav1.ListOptions{}) // List all priorityClasses in the vCluster
			if err != nil {
				return nil
			}
			var names []string
			for _, ic := range ics.Items {
				names = append(names, ic.Name)
			}
			return names
		}).WithTimeout(time.Minute).WithPolling(time.Second).
			Should(gomega.ContainElement(hpriorityClassName))

		gomega.Consistently(func() []string {
			ics, err := f.VClusterClient.SchedulingV1().PriorityClasses().List(f.Context, metav1.ListOptions{})
			if err != nil {
				return nil
			}
			var names []string
			for _, ic := range ics.Items {
				names = append(names, ic.Name)
			}
			return names
		}).WithTimeout(5 * time.Second).WithPolling(time.Second).
			ShouldNot(gomega.ContainElement(lpriorityClassName))
	})

	ginkgo.It("should get an error for pod creation using filtered priorityClass to host", func() {
		lpPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      lpPodName,
				Namespace: testNamespace,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: "nginx",
					},
				},
				PriorityClassName: lpriorityClassName,
			},
		}
		_, err := f.VClusterClient.CoreV1().Pods(testNamespace).Create(f.Context, lpPod, metav1.CreateOptions{})
		gomega.Expect(err).To(gomega.HaveOccurred())
	})

	ginkgo.It("should sync vcluster pod created with allowed priorityClass to host", func() {
		hpPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      hpPodName,
				Namespace: testNamespace,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: "nginx",
					},
				},
				PriorityClassName: hpriorityClassName,
			},
		}
		_, err := f.VClusterClient.CoreV1().Pods(testNamespace).Create(f.Context, hpPod, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		// Pod should be synced to host
		gomega.Eventually(func() []string {
			pods, err := f.HostClient.CoreV1().Pods(hostNamespace).List(f.Context, metav1.ListOptions{}) // List all pods in the vCluster
			if err != nil {
				return nil
			}
			var names []string
			for _, po := range pods.Items {
				names = append(names, po.Name)
			}
			return names
		}).WithTimeout(time.Minute).WithPolling(time.Second).
			Should(gomega.ContainElement(hpPodName + "-x-" + testNamespace + "-x-" + hostNamespace))

	})

})
