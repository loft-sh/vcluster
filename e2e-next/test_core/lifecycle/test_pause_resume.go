package lifecycle

import (
	"context"
	"os/exec"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/cli/find"
	"github.com/loft-sh/vcluster/pkg/util/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// PauseResumeSpec registers vcluster pause/resume tests against the framework-provisioned vcluster.
// All tests shell out to the vcluster binary (must be in $PATH or $GOBIN).
func PauseResumeSpec() {
	Describe("vCluster pause and resume",
		labels.CLI,
		func() {
			var (
				vClusterName      string
				vClusterNamespace string
				kubeContext       string
			)

			BeforeEach(func(ctx context.Context) context.Context {
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterNamespace = "vcluster-" + vClusterName
				kubeContext = "kind-" + constants.GetHostClusterName()
				return ctx
			})

			It("should pause the vcluster (scale down pods) and resume it (pods running again)", func(ctx context.Context) {
				hClient := cluster.KubeClientFrom(ctx, constants.GetHostClusterName())

				By("verifying vcluster pods are running before pause", func() {
					pods, err := hClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
						LabelSelector: "app=vcluster",
					})
					Expect(err).To(Succeed(), "listing vcluster pods in %s before pause", vClusterNamespace)
					Expect(pods.Items).ToNot(BeEmpty(), "expected at least one vcluster pod in %s before pause", vClusterNamespace)
				})

				// PollingTimeoutVeryLong because the CLI's internal pause path polls
				// the StatefulSet for up to 3 minutes before giving up.
				By("pausing the vcluster via CLI", func() {
					cmdCtx, cancel := context.WithTimeout(ctx, constants.PollingTimeoutVeryLong)
					defer cancel()
					cmd := exec.CommandContext(cmdCtx, vclusterBin(),
						"pause", vClusterName,
						"-n", vClusterNamespace,
						"--context", kubeContext,
					)
					out, err := cmd.CombinedOutput()
					Expect(err).To(Succeed(),
						"vcluster pause %s -n %s failed: %s", vClusterName, vClusterNamespace, string(out))
				})

				By("verifying vcluster pods are gone after pause", func() {
					Eventually(func(g Gomega) {
						pods, err := hClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
							LabelSelector: "app=vcluster",
						})
						g.Expect(err).To(Succeed(),
							"listing vcluster pods in %s after pause", vClusterNamespace)
						g.Expect(pods.Items).To(BeEmpty(),
							"expected zero vcluster pods in %s after pause, got %d",
							vClusterNamespace, len(pods.Items))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("verifying pause annotations are set on the StatefulSet", func() {
					sts, err := hClient.AppsV1().StatefulSets(vClusterNamespace).Get(ctx, vClusterName, metav1.GetOptions{})
					Expect(err).To(Succeed(), "get StatefulSet %s/%s", vClusterNamespace, vClusterName)
					Expect(sts.Annotations).To(HaveKeyWithValue("loft.sh/paused", "true"),
						"StatefulSet %s/%s should have loft.sh/paused=true after pause", vClusterNamespace, vClusterName)
				})

				By("resuming the vcluster via CLI", func() {
					cmdCtx, cancel := context.WithTimeout(ctx, constants.PollingTimeoutVeryLong)
					defer cancel()
					cmd := exec.CommandContext(cmdCtx, vclusterBin(),
						"resume", vClusterName,
						"-n", vClusterNamespace,
						"--context", kubeContext,
					)
					out, err := cmd.CombinedOutput()
					Expect(err).To(Succeed(),
						"vcluster resume %s -n %s failed: %s", vClusterName, vClusterNamespace, string(out))
				})

				By("waiting for vcluster pods to be running again after resume", func() {
					Eventually(func(g Gomega) {
						pods, err := hClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
							LabelSelector: "app=vcluster",
						})
						g.Expect(err).To(Succeed(),
							"listing vcluster pods in %s after resume", vClusterNamespace)
						g.Expect(pods.Items).NotTo(BeEmpty(),
							"expected vcluster pods in %s to reappear after resume", vClusterNamespace)
						for _, pod := range pods.Items {
							g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning),
								"pod %s in %s is not Running after resume (phase=%s)",
								pod.Name, vClusterNamespace, pod.Status.Phase)
							for _, cs := range pod.Status.ContainerStatuses {
								g.Expect(cs.Ready).To(BeTrue(),
									"container %s in pod %s is not ready after resume",
									cs.Name, pod.Name)
							}
						}
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
				})
			})
		},
	)
}

// PauseResumeScaledDownSpec registers pause/resume tests for a scaled-down tenant cluster.
// These tests create their own vcluster via the CLI (not framework-provisioned).
func PauseResumeScaledDownSpec() {
	Describe("pause and resume a scaled-down tenant cluster", labels.Core, labels.PR, Ordered, func() {
		// Ordered because each spec depends on the state from the prior spec:
		// create → scale down → pause → resume.
		var (
			suffix      string
			clusterName string
			namespace   string
			hostClient  kubernetes.Interface
		)

		BeforeAll(func(ctx context.Context) {
			suffix = random.String(6)
			clusterName = "e2e-pause-resume-sd-" + suffix
			namespace = clusterName
			hostClient = hostKubeClient()
			createAndWaitForReady(ctx, hostClient, clusterName, namespace)
			scaleDownVCluster(ctx, hostClient, clusterName, namespace)
		})

		AfterAll(func(ctx context.Context) {
			_, err := runVClusterCmd(ctx, "delete", clusterName, "-n", namespace, "--delete-namespace")
			Expect(err).To(Succeed())
		})

		It("should pause a scaled-down tenant cluster", func(ctx context.Context) {
			By("Pausing the scaled-down tenant cluster", func() {
				_, err := runVClusterCmd(ctx, "pause", clusterName, "-n", namespace)
				Expect(err).To(Succeed())
			})

			By("Verifying paused annotations are set on the StatefulSet", func() {
				sts, err := hostClient.AppsV1().StatefulSets(namespace).Get(ctx, clusterName, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(sts.Annotations).To(HaveKeyWithValue("loft.sh/paused", "true"),
					"StatefulSet %s/%s should have loft.sh/paused=true annotation", namespace, clusterName)
				Expect(sts.Annotations).To(HaveKeyWithValue("loft.sh/paused-replicas", "1"),
					"StatefulSet %s/%s should have loft.sh/paused-replicas=1 annotation", namespace, clusterName)
			})
		})

		It("should resume the tenant cluster after pause", func(ctx context.Context) {
			By("Resuming the tenant cluster", func() {
				_, err := runVClusterCmd(ctx, "resume", clusterName, "-n", namespace)
				Expect(err).To(Succeed())
			})

			By("Waiting for tenant cluster to be ready", func() {
				waitForVClusterReady(ctx, hostClient, clusterName, namespace)
			})

			By("Verifying StatefulSet has correct replica count", func() {
				sts, err := hostClient.AppsV1().StatefulSets(namespace).Get(ctx, clusterName, metav1.GetOptions{})
				Expect(err).To(Succeed())
				Expect(sts.Spec.Replicas).NotTo(BeNil())
				Expect(*sts.Spec.Replicas).To(Equal(int32(1)),
					"StatefulSet %s/%s should have 1 replica, got %d", namespace, clusterName, *sts.Spec.Replicas)
			})

			By("Verifying list shows Running status", func() {
				Eventually(func(g Gomega, ctx context.Context) {
					entries, err := listVClusters(ctx, namespace)
					g.Expect(err).To(Succeed())
					found := findByName(entries, clusterName)
					g.Expect(found).NotTo(BeNil(), "tenant cluster %s not found in list", clusterName)
					g.Expect(found.Status).To(Equal(string(find.StatusRunning)),
						"tenant cluster %s has status %s, expected %s", clusterName, found.Status, find.StatusRunning)
				}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
			})
		})
	})
}
