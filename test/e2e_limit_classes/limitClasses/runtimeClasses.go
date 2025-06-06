package limitclasses

import (
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	nodev1 "k8s.io/api/node/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Test limitclass on fromHost", ginkgo.Ordered, func() {
	var (
		f              *framework.Framework
		runcClassName  = "runc-runtimeclass"
		runscClassName = "runsc-runtimeclass"

		labelValue1 = "one"
		labelValue2 = "two"

		rcPodName  = "runc-pod"
		rscPodName = "runsc-pod"

		testNamespace = "default"
		hostNamespace = "vcluster"
	)

	ginkgo.BeforeAll(func() {
		f = framework.DefaultFramework
		ginkgo.By("Creating runc runtimeClass on host")
		runcclass := &nodev1.RuntimeClass{
			ObjectMeta: metav1.ObjectMeta{
				Name:   runcClassName,
				Labels: map[string]string{"value": labelValue1},
			},
			Handler: "runc",
		}
		_, err := f.HostClient.NodeV1().RuntimeClasses().Create(f.Context, runcclass, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		ginkgo.By("Creating runsc runtimeClass on host")
		runscclass := &nodev1.RuntimeClass{
			ObjectMeta: metav1.ObjectMeta{
				Name:   runscClassName,
				Labels: map[string]string{"value": labelValue2},
			},
			Handler: "runsc",
		}
		_, err = f.HostClient.NodeV1().RuntimeClasses().Create(f.Context, runscclass, metav1.CreateOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.AfterAll(func() {
		_ = f.HostClient.NodeV1().RuntimeClasses().Delete(f.Context, runcClassName, metav1.DeleteOptions{})
		_ = f.HostClient.NodeV1().RuntimeClasses().Delete(f.Context, runscClassName, metav1.DeleteOptions{})
		_ = f.HostClient.CoreV1().Pods(testNamespace).Delete(f.Context, rcPodName, metav1.DeleteOptions{})
		_ = f.HostClient.CoreV1().Pods(testNamespace).Delete(f.Context, rscPodName, metav1.DeleteOptions{})
	})

	ginkgo.It("should only sync runtimeClasses to virtual with allowed label", func() {
		ginkgo.By("Listing all runtimeClasses in the vCluster")
		rcs, err := f.VClusterClient.NodeV1().RuntimeClasses().List(f.Context, metav1.ListOptions{})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		var names []string
		for _, rc := range rcs.Items {
			names = append(names, rc.Name)
		}
		gomega.Expect(names).To(gomega.ContainElement(runcClassName))
		ginkgo.By("Found runc in vcluster")
		gomega.Expect(names).NotTo(gomega.ContainElement(runscClassName))
		ginkgo.By("runsc is not available in vcluster")
	})

	ginkgo.It("should get an error for pod creation using filtered runtimeClass to host", func() {
		ginkgo.By("Creating a pod using runsc runtimeClass in vcluster")
		runscpod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      rscPodName,
				Namespace: testNamespace,
			},
			Spec: corev1.PodSpec{
				RuntimeClassName: &runscClassName,
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: "nginx",
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 80,
							},
						},
					},
				},
			},
		}
		_, err := f.VClusterClient.CoreV1().Pods(testNamespace).Create(f.Context, runscpod, metav1.CreateOptions{})
		ginkgo.By("An error should be triggered")
		expectedSubstring := fmt.Sprintf(`pods "%s" is forbidden: pod rejected: RuntimeClass "%s" not found`, rscPodName, runscClassName)
		gomega.Expect(err).To(gomega.MatchError(gomega.ContainSubstring(expectedSubstring)))
	})

	ginkgo.It("should sync vcluster pod created with allowed runtimeClass to host", func() {
		ginkgo.By("Creating a pod using runc runtimeClass in vcluster")
		runcpod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      rcPodName,
				Namespace: testNamespace,
			},
			Spec: corev1.PodSpec{
				RuntimeClassName: &runcClassName,
				Containers: []corev1.Container{
					{
						Name:  "nginx",
						Image: "nginx",
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 80,
							},
						},
					},
				},
			},
		}
		_, err := f.VClusterClient.CoreV1().Pods(testNamespace).Create(f.Context, runcpod, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		ginkgo.By("Pod should be synced to host")
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
			Should(gomega.ContainElement(rcPodName + "-x-" + testNamespace + "-x-" + hostNamespace))

	})

})
