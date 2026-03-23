package coredns

import (
	"context"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/coredns"
	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	"github.com/loft-sh/vcluster/pkg/util/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
)

// DescribeCoreDNS registers CoreDNS resolution tests against the given vCluster.
func DescribeCoreDNS(vcluster suite.Dependency) bool {
	return Describe("CoreDNS resolves host names correctly",
		labels.Core,
		labels.CoreDNS,
		cluster.Use(vcluster),
		cluster.Use(clusters.HostCluster),
		func() {
			var (
				vClusterClient kubernetes.Interface
				vClusterConfig *rest.Config
			)

			BeforeEach(func(ctx context.Context) {
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				currentClusterName := cluster.CurrentClusterNameFrom(ctx)
				vClusterConfig = cluster.From(ctx, currentClusterName).KubernetesRestConfig()
				Expect(vClusterConfig).NotTo(BeNil())
			})

			It("should resolve a service via its hostname", func(ctx context.Context) {
				suffix := random.String(6)
				nsName := "coredns-svc-test-" + suffix

				_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: nsName},
				}, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				By("Creating a curl pod", func() {
					_, err := vClusterClient.CoreV1().Pods(nsName).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: "curl-" + suffix},
						Spec: corev1.PodSpec{
							TerminationGracePeriodSeconds: ptr.To(int64(1)),
							Containers: []corev1.Container{
								{
									Name:            "curl",
									Image:           "curlimages/curl",
									ImagePullPolicy: corev1.PullIfNotPresent,
									Command:         []string{"sleep"},
									Args:            []string{"9999"},
									SecurityContext: &corev1.SecurityContext{
										RunAsUser: ptr.To(int64(12345)),
									},
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				curlPodName := "curl-" + suffix

				By("Creating an nginx pod and service", func() {
					nginxLabels := map[string]string{"app": "nginx-" + suffix}
					_, err := vClusterClient.CoreV1().Pods(nsName).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name:   "nginx-" + suffix,
							Labels: nginxLabels,
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            "nginx",
									Image:           "nginxinc/nginx-unprivileged:stable-alpine3.20-slim",
									ImagePullPolicy: corev1.PullIfNotPresent,
									SecurityContext: &corev1.SecurityContext{
										RunAsUser: ptr.To(int64(12345)),
									},
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())

					_, err = vClusterClient.CoreV1().Services(nsName).Create(ctx, &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{Name: "nginx-" + suffix},
						Spec: corev1.ServiceSpec{
							Selector: nginxLabels,
							Ports:    []corev1.ServicePort{{Port: 8080}},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				By("Waiting for both pods to be running", func() {
					Eventually(func(g Gomega) {
						pod, err := vClusterClient.CoreV1().Pods(nsName).Get(ctx, curlPodName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning), "curl pod not yet running")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

					Eventually(func(g Gomega) {
						pod, err := vClusterClient.CoreV1().Pods(nsName).Get(ctx, "nginx-"+suffix, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning), "nginx pod not yet running")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Verifying the service is reachable via its DNS hostname", func() {
					url := fmt.Sprintf("http://nginx-%s.%s.svc:8080/", suffix, nsName)
					cmd := []string{"curl", "-s", "--show-error", "-o", "/dev/null", "-w", "%{http_code}", "--max-time", "5", url}
					Eventually(func(g Gomega) {
						stdout, _, err := podhelper.ExecBuffered(ctx, vClusterConfig, nsName, curlPodName, "curl", cmd, nil)
						g.Expect(err).NotTo(HaveOccurred(), "curl exec failed")
						g.Expect(string(stdout)).To(Equal("200"), "expected 200 from nginx service, got %s", string(stdout))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})

			It("should resolve fake kubelet endpoints via node hostnames", func(ctx context.Context) {
				suffix := random.String(6)
				nsName := "coredns-kubelet-test-" + suffix

				_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: nsName},
				}, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				By("Creating a curl pod", func() {
					_, err := vClusterClient.CoreV1().Pods(nsName).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: "curl-" + suffix},
						Spec: corev1.PodSpec{
							TerminationGracePeriodSeconds: ptr.To(int64(1)),
							Containers: []corev1.Container{
								{
									Name:            "curl",
									Image:           "curlimages/curl",
									ImagePullPolicy: corev1.PullIfNotPresent,
									Command:         []string{"sleep"},
									Args:            []string{"9999"},
									SecurityContext: &corev1.SecurityContext{
										RunAsUser: ptr.To(int64(12345)),
									},
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				curlPodName := "curl-" + suffix

				By("Waiting for the curl pod to be running", func() {
					Eventually(func(g Gomega) {
						pod, err := vClusterClient.CoreV1().Pods(nsName).Get(ctx, curlPodName, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning), "curl pod not yet running")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Checking each node's kubelet /healthz endpoint is reachable", func() {
					nodes, err := vClusterClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
					Expect(err).To(Succeed())
					Expect(nodes.Items).NotTo(BeEmpty(), "expected at least one node in vCluster")

					for _, node := range nodes.Items {
						hostname := node.Name
						for _, address := range node.Status.Addresses {
							if address.Type == corev1.NodeHostName {
								hostname = address.Address
								break
							}
						}

						url := fmt.Sprintf("https://%s:%d/healthz", hostname, node.Status.DaemonEndpoints.KubeletEndpoint.Port)
						cmd := []string{"curl", "-k", "-s", "--show-error", url}
						// CoreDNS reloads NodeHosts every 15s; retry until the hostname resolves
						Eventually(func(g Gomega) {
							stdout, stderr, err := podhelper.ExecBuffered(ctx, vClusterConfig, nsName, curlPodName, "curl", cmd, nil)
							g.Expect(err).NotTo(HaveOccurred(), "curl to kubelet failed for node %s: stderr=%s", node.Name, string(stderr))
							g.Expect(string(stderr)).To(BeEmpty(), "unexpected stderr from kubelet healthz on node %s", node.Name)
							g.Expect(string(stdout)).To(Equal("ok"), "expected 'ok' from kubelet healthz on node %s, got %q", node.Name, string(stdout))
						}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
					}
				})
			})

			It("should use the pinned CoreDNS image version", func(ctx context.Context) {
				By("Checking the CoreDNS deployment image", func() {
					coreDNSDeployment, err := vClusterClient.AppsV1().Deployments("kube-system").Get(ctx, "coredns", metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(coreDNSDeployment.Spec.Template.Spec.Containers).To(HaveLen(1))
					Expect(coreDNSDeployment.Spec.Template.Spec.Containers[0].Image).To(Equal(coredns.DefaultImage),
						"CoreDNS image should match the pinned default image")
					// Ensure we are not using images with known security vulnerabilities
					Expect(coreDNSDeployment.Spec.Template.Spec.Containers[0].Image).NotTo(ContainSubstring("1.11.1"))
					Expect(coreDNSDeployment.Spec.Template.Spec.Containers[0].Image).NotTo(ContainSubstring("1.11.0"))
				})
			})
		},
	)
}
