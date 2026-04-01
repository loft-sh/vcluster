package kubeletproxy

import (
	"encoding/json"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Kubelet proxy subpath access control", ginkgo.Ordered, func() {
	var f *framework.Framework

	ginkgo.BeforeAll(func() {
		f = framework.DefaultFramework
	})

	ginkgo.It("GET /healthz via node proxy returns ok for every virtual node", func() {
		var nodes *corev1.NodeList

		ginkgo.By("waiting for at least one virtual node to be registered")
		gomega.Eventually(func() bool {
			var err error
			nodes, err = f.VClusterClient.CoreV1().Nodes().List(f.Context, metav1.ListOptions{})
			if err != nil {
				return false
			}
			return len(nodes.Items) > 0
		}).WithPolling(framework.PollInterval).WithTimeout(framework.PollTimeout).Should(gomega.BeTrue(), "expected at least one virtual node")

		ginkgo.By("checking /healthz returns ok for each node")
		for _, node := range nodes.Items {
			data, err := f.VClusterClient.RESTClient().Get().
				AbsPath(fmt.Sprintf("/api/v1/nodes/%s/proxy/healthz", node.Name)).
				DoRaw(f.Context)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "GET /healthz should succeed for node %s", node.Name)
			gomega.Expect(string(data)).To(gomega.Equal("ok"), "GET /healthz should return ok for node %s", node.Name)
		}
	})

	ginkgo.It("GET /pods via node proxy returns only pods belonging to this vcluster", func() {
		nodes, err := f.VClusterClient.CoreV1().Nodes().List(f.Context, metav1.ListOptions{})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(nodes.Items).NotTo(gomega.BeEmpty())

		ginkgo.By("collecting all virtual pod identities")
		virtualPods, err := f.VClusterClient.CoreV1().Pods("").List(f.Context, metav1.ListOptions{})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		virtualPodKeys := make(map[string]bool, len(virtualPods.Items))
		for _, p := range virtualPods.Items {
			virtualPodKeys[p.Namespace+"/"+p.Name] = true
		}

		ginkgo.By("asserting every pod in the kubelet /pods response is a virtual pod")
		for _, node := range nodes.Items {
			data, err := f.VClusterClient.RESTClient().Get().
				AbsPath(fmt.Sprintf("/api/v1/nodes/%s/proxy/pods", node.Name)).
				DoRaw(f.Context)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "GET /pods should succeed for node %s", node.Name)

			podList := &corev1.PodList{}
			gomega.Expect(json.Unmarshal(data, podList)).To(gomega.Succeed())

			for _, pod := range podList.Items {
				key := pod.Namespace + "/" + pod.Name
				gomega.Expect(virtualPodKeys).To(gomega.HaveKey(key),
					"kubelet /pods response contains pod %q which is not in this virtual cluster — cross-tenant leak detected", key)
			}
		}
	})

	ginkgo.It("GET /runningpods via node proxy returns only pods belonging to this vcluster", func() {
		nodes, err := f.VClusterClient.CoreV1().Nodes().List(f.Context, metav1.ListOptions{})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(nodes.Items).NotTo(gomega.BeEmpty())

		virtualPods, err := f.VClusterClient.CoreV1().Pods("").List(f.Context, metav1.ListOptions{})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		virtualPodKeys := make(map[string]bool, len(virtualPods.Items))
		for _, p := range virtualPods.Items {
			virtualPodKeys[p.Namespace+"/"+p.Name] = true
		}

		ginkgo.By("asserting every pod in the kubelet /runningpods response is a virtual pod")
		for _, node := range nodes.Items {
			data, err := f.VClusterClient.RESTClient().Get().
				AbsPath(fmt.Sprintf("/api/v1/nodes/%s/proxy/runningpods", node.Name)).
				DoRaw(f.Context)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "GET /runningpods should succeed for node %s", node.Name)

			podList := &corev1.PodList{}
			gomega.Expect(json.Unmarshal(data, podList)).To(gomega.Succeed())

			for _, pod := range podList.Items {
				key := pod.Namespace + "/" + pod.Name
				gomega.Expect(virtualPodKeys).To(gomega.HaveKey(key),
					"kubelet /runningpods response contains pod %q which is not in this virtual cluster — cross-tenant leak detected", key)
			}
		}
	})

	ginkgo.It("GET /containerLogs for a non-existent pod returns 403 Forbidden", func() {
		nodes, err := f.VClusterClient.CoreV1().Nodes().List(f.Context, metav1.ListOptions{})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(nodes.Items).NotTo(gomega.BeEmpty())

		ginkgo.By("requesting container logs for a pod that does not exist in the virtual cluster")
		_, err = f.VClusterClient.RESTClient().Get().
			AbsPath(fmt.Sprintf("/api/v1/nodes/%s/proxy/containerLogs/nonexistent-ns/nonexistent-pod/main",
				nodes.Items[0].Name)).
			DoRaw(f.Context)
		gomega.Expect(err).To(gomega.HaveOccurred(), "expected an error for a non-existent pod")
		gomega.Expect(kerrors.IsForbidden(err)).To(gomega.BeTrue(),
			"expected 403 Forbidden, got: %v", err)
	})

	// The following two specs share a running pod, so they run sequentially in an Ordered
	// context. BeforeAll creates the pod once; both It blocks depend on it existing.
	ginkgo.Context("with a running test pod", ginkgo.Ordered, func() {
		var (
			nsName   string
			podName  string
			nodeName string
		)

		ginkgo.BeforeAll(func() {
			suffix := random.String(6)
			nsName = "kubelet-proxy-test-" + suffix
			podName = "logger-" + suffix

			ginkgo.By("creating the test namespace")
			_, err := f.VClusterClient.CoreV1().Namespaces().Create(f.Context,
				&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}},
				metav1.CreateOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("creating a pod that writes a known log line")
			_, err = f.VClusterClient.CoreV1().Pods(nsName).Create(f.Context, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: nsName},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:    "logger",
							Image:   "busybox",
							Command: []string{"sh", "-c", "echo kubelet-proxy-test-ok && sleep 3600"},
						},
					},
				},
			}, metav1.CreateOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("waiting for the pod to reach Running phase")
			gomega.Eventually(func() bool {
				pod, err := f.VClusterClient.CoreV1().Pods(nsName).Get(f.Context, podName, metav1.GetOptions{})
				if err != nil {
					return false
				}
				return pod.Status.Phase == corev1.PodRunning
			}).WithPolling(framework.PollInterval).WithTimeout(framework.PollTimeoutLong).Should(gomega.BeTrue(),
				"timed out waiting for pod to reach Running phase")

			ginkgo.By("capturing the node the pod was scheduled to")
			pod, err := f.VClusterClient.CoreV1().Pods(nsName).Get(f.Context, podName, metav1.GetOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			nodeName = pod.Spec.NodeName
			gomega.Expect(nodeName).NotTo(gomega.BeEmpty(), "pod should be scheduled to a node")
		})

		ginkgo.AfterAll(func() {
			_ = f.VClusterClient.CoreV1().Namespaces().Delete(f.Context, nsName, metav1.DeleteOptions{})
		})

		ginkgo.It("GET /containerLogs with a valid pod and container returns log output", func() {
			ginkgo.By("fetching container logs via the kubelet proxy")
			data, err := f.VClusterClient.RESTClient().Get().
				AbsPath(fmt.Sprintf("/api/v1/nodes/%s/proxy/containerLogs/%s/%s/logger",
					nodeName, nsName, podName)).
				DoRaw(f.Context)
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "GET /containerLogs should succeed for a running pod")
			gomega.Expect(string(data)).To(gomega.ContainSubstring("kubelet-proxy-test-ok"),
				"expected log output to contain the known log line")
		})

		ginkgo.It("GET /containerLogs with a valid pod but an unknown container returns 403 Forbidden", func() {
			// "injected-sidecar" simulates a host-only sidecar injected by an admission webhook
			// after the pod left the virtual cluster API server. The allowlist check must deny
			// access to containers absent from the virtual pod spec.
			ginkgo.By("requesting logs for a container name absent from the virtual pod spec")
			_, err := f.VClusterClient.RESTClient().Get().
				AbsPath(fmt.Sprintf("/api/v1/nodes/%s/proxy/containerLogs/%s/%s/injected-sidecar",
					nodeName, nsName, podName)).
				DoRaw(f.Context)
			gomega.Expect(err).To(gomega.HaveOccurred(), "expected 403 for a container not in the virtual pod spec")
			gomega.Expect(kerrors.IsForbidden(err)).To(gomega.BeTrue(),
				"expected 403 Forbidden, got: %v", err)
		})
	})
})
