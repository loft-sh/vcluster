package test_core

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DescribeKubeletProxy registers kubelet proxy subpath access control tests against the given vCluster.
func DescribeKubeletProxy(vcluster suite.Dependency) bool {
	return Describe("Kubelet proxy subpath access control",
		labels.Core,
		labels.Security,
		labels.PR,
		cluster.Use(vcluster),
		func() {
			var vClusterClientset *kubernetes.Clientset

			BeforeEach(func(ctx context.Context) context.Context {
				cfg := cluster.CurrentClusterFrom(ctx).KubernetesRestConfig()
				var err error
				vClusterClientset, err = kubernetes.NewForConfig(cfg)
				Expect(err).NotTo(HaveOccurred())
				return ctx
			})

			It("GET /healthz via node proxy returns ok for every virtual node", func(ctx context.Context) {
				var nodes *corev1.NodeList
				By("waiting for at least one virtual node to be registered", func() {
					Eventually(func(g Gomega) {
						var err error
						nodes, err = vClusterClientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
						g.Expect(err).NotTo(HaveOccurred())
						g.Expect(nodes.Items).NotTo(BeEmpty(), "expected at least one virtual node")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				By("checking /healthz returns ok for each node", func() {
					for _, node := range nodes.Items {
						data, err := vClusterClientset.RESTClient().Get().
							AbsPath(fmt.Sprintf("/api/v1/nodes/%s/proxy/healthz", node.Name)).
							DoRaw(ctx)
						Expect(err).NotTo(HaveOccurred(), "GET /healthz should succeed for node %s", node.Name)
						Expect(string(data)).To(Equal("ok"), "GET /healthz should return ok for node %s", node.Name)
					}
				})
			})

			It("GET /pods via node proxy returns only pods belonging to this vcluster", func(ctx context.Context) {
				nodes, err := vClusterClientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(nodes.Items).NotTo(BeEmpty())

				By("collecting all virtual pod identities", func() {})

				virtualPods, err := vClusterClientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				virtualPodKeys := make(map[string]bool, len(virtualPods.Items))
				for _, p := range virtualPods.Items {
					virtualPodKeys[p.Namespace+"/"+p.Name] = true
				}

				By("asserting every pod in the kubelet /pods response is a virtual pod", func() {
					for _, node := range nodes.Items {
						data, err := vClusterClientset.RESTClient().Get().
							AbsPath(fmt.Sprintf("/api/v1/nodes/%s/proxy/pods", node.Name)).
							DoRaw(ctx)
						Expect(err).NotTo(HaveOccurred(), "GET /pods should succeed for node %s", node.Name)

						podList := &corev1.PodList{}
						Expect(json.Unmarshal(data, podList)).To(Succeed())

						for _, pod := range podList.Items {
							key := pod.Namespace + "/" + pod.Name
							Expect(virtualPodKeys).To(HaveKey(key),
								"kubelet /pods response contains pod %q which is not in this virtual cluster — cross-tenant leak detected", key)
						}
					}
				})
			})

			It("GET /runningpods via node proxy returns only pods belonging to this vcluster", func(ctx context.Context) {
				nodes, err := vClusterClientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(nodes.Items).NotTo(BeEmpty())

				virtualPods, err := vClusterClientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				virtualPodKeys := make(map[string]bool, len(virtualPods.Items))
				for _, p := range virtualPods.Items {
					virtualPodKeys[p.Namespace+"/"+p.Name] = true
				}

				By("asserting every pod in the kubelet /runningpods response is a virtual pod", func() {
					for _, node := range nodes.Items {
						data, err := vClusterClientset.RESTClient().Get().
							AbsPath(fmt.Sprintf("/api/v1/nodes/%s/proxy/runningpods", node.Name)).
							DoRaw(ctx)
						Expect(err).NotTo(HaveOccurred(), "GET /runningpods should succeed for node %s", node.Name)

						podList := &corev1.PodList{}
						Expect(json.Unmarshal(data, podList)).To(Succeed())

						for _, pod := range podList.Items {
							key := pod.Namespace + "/" + pod.Name
							Expect(virtualPodKeys).To(HaveKey(key),
								"kubelet /runningpods response contains pod %q which is not in this virtual cluster — cross-tenant leak detected", key)
						}
					}
				})
			})

			It("GET /containerLogs for a non-existent pod returns 403 Forbidden", func(ctx context.Context) {
				nodes, err := vClusterClientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(nodes.Items).NotTo(BeEmpty())

				By("requesting container logs for a pod that does not exist in the virtual cluster", func() {
					_, err := vClusterClientset.RESTClient().Get().
						AbsPath(fmt.Sprintf("/api/v1/nodes/%s/proxy/containerLogs/nonexistent-ns/nonexistent-pod/main",
							nodes.Items[0].Name)).
						DoRaw(ctx)
					Expect(err).To(HaveOccurred(), "expected an error for a non-existent pod")
					Expect(kerrors.IsForbidden(err)).To(BeTrue(),
						"expected 403 Forbidden, got: %v", err)
				})
			})

			// The following two specs share a running pod, so they run sequentially in an Ordered
			// context. BeforeAll creates the pod once; both It blocks depend on it existing.
			Context("with a running test pod", Ordered, func() {
				var (
					nsName   string
					podName  string
					nodeName string
				)

				BeforeAll(func(ctx context.Context) {
					suffix := random.String(6)
					nsName = "kubelet-proxy-test-" + suffix
					podName = "logger-" + suffix

					By("creating the test namespace", func() {
						_, err := vClusterClientset.CoreV1().Namespaces().Create(ctx,
							&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}},
							metav1.CreateOptions{})
						Expect(err).NotTo(HaveOccurred())
					})
					DeferCleanup(func(ctx context.Context) {
						err := vClusterClientset.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).NotTo(HaveOccurred())
						}
					})

					By("creating a pod that writes a known log line", func() {
						_, err := vClusterClientset.CoreV1().Pods(nsName).Create(ctx, &corev1.Pod{
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
						Expect(err).NotTo(HaveOccurred())
					})

					By("waiting for the pod to reach Running phase", func() {
						Eventually(func(g Gomega) {
							pod, err := vClusterClientset.CoreV1().Pods(nsName).Get(ctx, podName, metav1.GetOptions{})
							g.Expect(err).NotTo(HaveOccurred())
							g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning),
								"pod phase: %s, conditions: %v", pod.Status.Phase, pod.Status.Conditions)
						}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
					})

					By("capturing the node the pod was scheduled to", func() {
						pod, err := vClusterClientset.CoreV1().Pods(nsName).Get(ctx, podName, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						nodeName = pod.Spec.NodeName
						Expect(nodeName).NotTo(BeEmpty(), "pod should be scheduled to a node")
					})
				})

				It("GET /containerLogs with a valid pod and container returns log output", func(ctx context.Context) {
					By("fetching container logs via the kubelet proxy", func() {
						data, err := vClusterClientset.RESTClient().Get().
							AbsPath(fmt.Sprintf("/api/v1/nodes/%s/proxy/containerLogs/%s/%s/logger",
								nodeName, nsName, podName)).
							DoRaw(ctx)
						Expect(err).NotTo(HaveOccurred(), "GET /containerLogs should succeed for a running pod")
						Expect(string(data)).To(ContainSubstring("kubelet-proxy-test-ok"),
							"expected log output to contain the known log line")
					})
				})

				It("GET /containerLogs with a valid pod but an unknown container returns 403 Forbidden", func(ctx context.Context) {
					// "injected-sidecar" simulates a host-only sidecar injected by an admission webhook
					// after the pod left the virtual cluster API server. The allowlist check must deny
					// access to containers absent from the virtual pod spec.
					By("requesting logs for a container name absent from the virtual pod spec", func() {
						_, err := vClusterClientset.RESTClient().Get().
							AbsPath(fmt.Sprintf("/api/v1/nodes/%s/proxy/containerLogs/%s/%s/injected-sidecar",
								nodeName, nsName, podName)).
							DoRaw(ctx)
						Expect(err).To(HaveOccurred(), "expected 403 for a container not in the virtual pod spec")
						Expect(kerrors.IsForbidden(err)).To(BeTrue(),
							"expected 403 Forbidden, got: %v", err)
					})
				})
			})
		},
	)
}
