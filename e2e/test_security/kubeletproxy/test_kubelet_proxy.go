// Package kubeletproxy contains kubelet proxy access control tests.
package kubeletproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e/constants"
	"github.com/loft-sh/vcluster/e2e/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// KubeletProxySpec registers the spec.
func KubeletProxySpec() {
	Describe("Kubelet proxy subpath access control",
		labels.Security,
		func() {
			var vClusterClientset *kubernetes.Clientset

			BeforeEach(func(ctx context.Context) context.Context {
				cfg := cluster.CurrentClusterFrom(ctx).KubernetesRestConfig()
				var err error
				vClusterClientset, err = kubernetes.NewForConfig(cfg)
				Expect(err).To(Succeed())
				return ctx
			})

			It("GET /healthz via node proxy returns ok for every virtual node", func(ctx context.Context) {
				var nodes *corev1.NodeList
				By("waiting for at least one virtual node to be registered", func() {
					Eventually(func(g Gomega) {
						var err error
						nodes, err = vClusterClientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
						g.Expect(err).To(Succeed(), "failed to list virtual nodes")
						g.Expect(nodes.Items).NotTo(BeEmpty(), "expected at least one virtual node")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				By("checking /healthz returns ok for each node", func() {
					for _, node := range nodes.Items {
						data, err := vClusterClientset.RESTClient().Get().
							AbsPath(fmt.Sprintf("/api/v1/nodes/%s/proxy/healthz", node.Name)).
							DoRaw(ctx)
						Expect(err).To(Succeed(), "GET /healthz should succeed for node %s", node.Name)
						Expect(string(data)).To(Equal("ok"), "GET /healthz should return ok for node %s", node.Name)
					}
				})
			})

			It("GET /pods via node proxy returns only pods belonging to this vcluster", func(ctx context.Context) {
				nodes, err := vClusterClientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
				Expect(err).To(Succeed())
				Expect(nodes.Items).NotTo(BeEmpty())

				var virtualPodKeys map[string]bool
				By("collecting all virtual pod identities", func() {
					virtualPods, err := vClusterClientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
					Expect(err).To(Succeed())
					virtualPodKeys = make(map[string]bool, len(virtualPods.Items))
					for _, p := range virtualPods.Items {
						virtualPodKeys[p.Namespace+"/"+p.Name] = true
					}
				})

				By("asserting every pod in the kubelet /pods response is a virtual pod", func() {
					for _, node := range nodes.Items {
						data, err := vClusterClientset.RESTClient().Get().
							AbsPath(fmt.Sprintf("/api/v1/nodes/%s/proxy/pods", node.Name)).
							DoRaw(ctx)
						Expect(err).To(Succeed(), "GET /pods should succeed for node %s", node.Name)

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
				Expect(err).To(Succeed())
				Expect(nodes.Items).NotTo(BeEmpty())

				virtualPods, err := vClusterClientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
				Expect(err).To(Succeed())
				virtualPodKeys := make(map[string]bool, len(virtualPods.Items))
				for _, p := range virtualPods.Items {
					virtualPodKeys[p.Namespace+"/"+p.Name] = true
				}

				By("asserting every pod in the kubelet /runningpods response is a virtual pod", func() {
					for _, node := range nodes.Items {
						data, err := vClusterClientset.RESTClient().Get().
							AbsPath(fmt.Sprintf("/api/v1/nodes/%s/proxy/runningpods", node.Name)).
							DoRaw(ctx)
						Expect(err).To(Succeed(), "GET /runningpods should succeed for node %s", node.Name)

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
				Expect(err).To(Succeed())
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
					// Ensure clientset is initialized. The outer BeforeEach sets it
					// for non-Ordered specs, but in an Ordered context the BeforeAll
					// runs before the first BeforeEach on some Ginkgo parallel nodes.
					// If the cluster was not provisioned (label filter mismatch), the
					// framework skips BeforeEach, so we must guard against nil here.
					if vClusterClientset == nil {
						currentCluster := cluster.CurrentClusterFrom(ctx)
						if currentCluster == nil {
							Skip("kubelet-proxy-vcluster not provisioned for this label filter")
						}
						cfg := currentCluster.KubernetesRestConfig()
						var err error
						vClusterClientset, err = kubernetes.NewForConfig(cfg)
						Expect(err).To(Succeed())
					}

					suffix := random.String(6)
					nsName = "kubelet-proxy-test-" + suffix
					podName = "logger-" + suffix

					_, err := vClusterClientset.CoreV1().Namespaces().Create(ctx,
						&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}},
						metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := vClusterClientset.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).To(Succeed())
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
						Expect(err).To(Succeed())
					})

					By("waiting for the pod to reach Running phase", func() {
						Eventually(func(g Gomega) {
							pod, err := vClusterClientset.CoreV1().Pods(nsName).Get(ctx, podName, metav1.GetOptions{})
							g.Expect(err).To(Succeed(), "failed to get pod %s/%s", nsName, podName)
							g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning),
								"pod phase: %s, conditions: %v", pod.Status.Phase, pod.Status.Conditions)
						}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
					})

					By("capturing the node the pod was scheduled to", func() {
						pod, err := vClusterClientset.CoreV1().Pods(nsName).Get(ctx, podName, metav1.GetOptions{})
						Expect(err).To(Succeed())
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
						Expect(err).To(Succeed(), "GET /containerLogs should succeed for a running pod")
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

				// client-go cleans "." and ".." out of AbsPath, and the vcluster's ServeMux redirects
				// literal dot segments to the cleaned path, so these specs send the traversal with
				// encoded slashes over raw HTTP.
				It("GET /containerLogs for an owned pod with an encoded traversal to /pods returns 400 Bad Request", func(ctx context.Context) {
					By("requesting container logs that walk to the unfiltered kubelet /pods", func() {
						expectEncodedTraversalRejected(ctx, nodeName, nsName, podName, "%2F..%2F..%2F..%2F..%2Fpods")
					})
				})

				It("GET /containerLogs for an owned pod with an encoded traversal to /configz returns 400 Bad Request", func(ctx context.Context) {
					By("requesting container logs that walk to the kubelet /configz", func() {
						expectEncodedTraversalRejected(ctx, nodeName, nsName, podName, "%2F..%2F..%2F..%2F..%2Fconfigz")
					})
				})

				It("GET /containerLogs for an owned pod with an encoded traversal to another pod's logs returns 400 Bad Request", func(ctx context.Context) {
					By("requesting container logs that walk to another pod's containerLogs", func() {
						expectEncodedTraversalRejected(ctx, nodeName, nsName, podName,
							"%2F..%2F..%2F..%2Fkube-system%2Fother-pod%2Fother-container")
					})
				})

				It("GET /containerLogs for an owned pod with literal traversal segments is redirected to the cleaned path", func(ctx context.Context) {
					By("requesting container logs with literal dot segments after the owned container", func() {
						rawPath := fmt.Sprintf("/api/v1/nodes/%s/proxy/containerLogs/%s/%s/logger/../../../../configz",
							nodeName, nsName, podName)
						code, body := getRawPath(ctx, rawPath)
						Expect(code).To(Equal(http.StatusTemporaryRedirect),
							"expected the ServeMux to redirect %q, got %d: %s", rawPath, code, body)
					})
				})

				It("GET /containerLogs for an owned pod with an extra trailing segment returns 403 Forbidden", func(ctx context.Context) {
					By("requesting container logs with a segment after the owned container", func() {
						_, err := vClusterClientset.RESTClient().Get().
							AbsPath(fmt.Sprintf("/api/v1/nodes/%s/proxy/containerLogs/%s/%s/logger/extra",
								nodeName, nsName, podName)).
							DoRaw(ctx)
						Expect(err).To(HaveOccurred(), "expected 403 for a containerLogs path with an extra segment")
						Expect(kerrors.IsForbidden(err)).To(BeTrue(),
							"expected 403 Forbidden, got: %v", err)
					})
				})
			})
		},
	)
}

// expectEncodedTraversalRejected requests the logs of the owned logger container with suffix
// appended after the container name, and expects the vcluster to reject the dot segments.
func expectEncodedTraversalRejected(ctx context.Context, nodeName, nsName, podName, suffix string) {
	GinkgoHelper()

	rawPath := fmt.Sprintf("/api/v1/nodes/%s/proxy/containerLogs/%s/%s/logger%s",
		nodeName, nsName, podName, suffix)
	code, body := getRawPath(ctx, rawPath)
	Expect(code).To(Equal(http.StatusBadRequest),
		"expected 400 for %q, got %d: %s", rawPath, code, body)
	Expect(body).To(ContainSubstring("must not contain '.' or '..' segments"),
		"unexpected response body for %q", rawPath)
}

// getRawPath sends a GET for rawPath to the current vcluster without cleaning or re-encoding the
// path, and returns the status code and response body.
func getRawPath(ctx context.Context, rawPath string) (int, string) {
	GinkgoHelper()

	cfg := cluster.CurrentClusterFrom(ctx).KubernetesRestConfig()
	httpClient, err := rest.HTTPClientFor(cfg)
	Expect(err).To(Succeed())
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSuffix(cfg.Host, "/")+rawPath, nil)
	Expect(err).To(Succeed())
	Expect(req.URL.EscapedPath()).To(HaveSuffix(rawPath), "request path was cleaned or re-encoded before sending")

	resp, err := httpClient.Do(req)
	Expect(err).To(Succeed())
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	Expect(err).To(Succeed())
	return resp.StatusCode, string(body)
}
