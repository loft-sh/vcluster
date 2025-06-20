package limitclasses

import (
	"fmt"
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
		ginkgo.By("Creating high-priority priorityClass on host")
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

		ginkgo.By("Creating low-priority priorityClass on host")
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
		ginkgo.By("Listing all priorityClasses in vcluster")
		gomega.Eventually(func() []string {
			pcs, err := f.VClusterClient.SchedulingV1().PriorityClasses().List(f.Context, metav1.ListOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			var names []string
			for _, pc := range pcs.Items {
				names = append(names, pc.Name)
			}
			return names
		}).WithTimeout(time.Minute).WithPolling(time.Second).Should(gomega.ContainElement(hpriorityClassName))

		ginkgo.By("Found high-priority in vcluster")

		gomega.Consistently(func() []string {
			pcs, err := f.VClusterClient.SchedulingV1().PriorityClasses().List(f.Context, metav1.ListOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			var names []string
			for _, pc := range pcs.Items {
				names = append(names, pc.Name)
			}
			return names
		}).WithTimeout(time.Minute).WithPolling(time.Second).ShouldNot(gomega.ContainElement(lpriorityClassName))

		ginkgo.By("low-priority is not available in vcluster")
	})

	ginkgo.It("should get an error for pod creation using filtered priorityClass to host", func() {
		ginkgo.By("Creating a pod using low-prority priorityClass in vcluster")
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
		ginkgo.By("An error should be triggered")
		expectedSubstring := fmt.Sprintf(`pods "%s" is forbidden: no PriorityClass with name %s was found`, lpPodName, lpriorityClassName)
		gomega.Expect(err).To(gomega.MatchError(gomega.ContainSubstring(expectedSubstring)))
	})

	ginkgo.It("should sync vcluster pod created with allowed priorityClass to host", func() {
		ginkgo.By("Creating a pod using high-prority priorityClass in vcluster")
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

		ginkgo.By("Pod should be synced to host")
		ginkgo.By("Listing all pods in host's vcluster namespace")
		gomega.Eventually(func() []string {
			pods, err := f.HostClient.CoreV1().Pods(hostNamespace).List(f.Context, metav1.ListOptions{})
			if err != nil {
				return nil
			}
			var names []string
			for _, pod := range pods.Items {
				names = append(names, pod.Name)
			}
			return names
		}).WithTimeout(time.Minute).WithPolling(time.Second).
			Should(gomega.ContainElement(hpPodName + "-x-" + testNamespace + "-x-" + hostNamespace))
	})

})
