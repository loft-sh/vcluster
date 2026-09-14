package metricsproxy

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	loftlog "github.com/loft-sh/log"
	connectcmd "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"
	"github.com/loft-sh/vcluster/e2e/constants"
	"github.com/loft-sh/vcluster/e2e/labels"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/util/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientpkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// MetricsProxyRestartSpec registers the control plane restart test for the
// metrics proxy integration. On every start the syncer re-applies the
// deletion protection policy for the metrics APIService. Once the policy
// exists that apply changes nothing, and the control plane used to crash-loop
// waiting for a cache update that never came (ENGCP-1458).
//
// The spec deletes the control plane pod, so the suite connection is dead
// afterwards. Register it after the other metrics proxy specs.
func MetricsProxyRestartSpec() {
	Describe("Metrics proxy integration after a control plane restart",
		labels.Integration,
		func() {
			var (
				vClusterName      string
				vClusterNamespace string
				hostKubeconfig    string
				hostClient        kubernetes.Interface
				vClusterClient    kubernetes.Interface
			)

			BeforeEach(func(ctx context.Context) context.Context {
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterNamespace = "vcluster-" + vClusterName
				hostKubeconfig = cluster.From(ctx, constants.GetHostClusterName()).GetKubeconfig()
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				return ctx
			})

			It("should come back with the metrics API available after the control plane pod is deleted", func(ctx context.Context) {
				suffix := random.String(6)
				workloadNamespace := "metrics-restart-" + suffix
				workloadPodName := "metrics-restart-workload-" + suffix

				By("Running a workload so the control plane restarts on a non-empty tenant cluster", func() {
					_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: workloadNamespace},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed(), "creating namespace %s", workloadNamespace)
					DeferCleanup(func(ctx context.Context) {
						// vClusterClient is replaced by a fresh connection after the restart
						err := vClusterClient.CoreV1().Namespaces().Delete(ctx, workloadNamespace, metav1.DeleteOptions{})
						Expect(clientpkg.IgnoreNotFound(err)).To(Succeed(), "deleting namespace %s", workloadNamespace)
					})

					_, err = vClusterClient.CoreV1().Pods(workloadNamespace).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: workloadPodName},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{Name: "nginx", Image: "nginx:1.25.0"}},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed(), "creating pod %s/%s", workloadNamespace, workloadPodName)

					Eventually(func(g Gomega) {
						pod, err := vClusterClient.CoreV1().Pods(workloadNamespace).Get(ctx, workloadPodName, metav1.GetOptions{})
						g.Expect(err).To(Succeed(), "getting pod %s/%s", workloadNamespace, workloadPodName)
						g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning),
							"pod %s/%s is %s, expected Running", workloadNamespace, workloadPodName, pod.Status.Phase)
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				var deletedPodUID types.UID
				By("Deleting the control plane pod", func() {
					pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
						LabelSelector: "app=vcluster,release=" + vClusterName,
					})
					Expect(err).To(Succeed(), "listing control plane pods in %s", vClusterNamespace)
					Expect(pods.Items).To(HaveLen(1), "expected exactly one control plane pod in %s, got %d", vClusterNamespace, len(pods.Items))

					deletedPodUID = pods.Items[0].UID
					err = hostClient.CoreV1().Pods(vClusterNamespace).Delete(ctx, pods.Items[0].Name, metav1.DeleteOptions{})
					Expect(err).To(Succeed(), "deleting control plane pod %s/%s", vClusterNamespace, pods.Items[0].Name)
				})

				By("Waiting for the replacement control plane pod to become ready without restarting", func() {
					waitForControlPlaneReplacement(ctx, hostClient, vClusterNamespace, vClusterName, deletedPodUID)
				})

				var vClusterConfig *rest.Config
				By("Reconnecting to the tenant cluster", func() {
					vClusterConfig = reconnectVCluster(ctx, vClusterName, vClusterNamespace, hostKubeconfig)
					vClusterClient = kubernetes.NewForConfigOrDie(vClusterConfig)
				})

				By("Waiting for the metrics API service to be available again", func() {
					waitForMetricsAPIServiceAvailable(ctx, vClusterConfig)
				})

				By("Listing node metrics through the restarted control plane", func() {
					waitForNodeMetrics(ctx, vClusterConfig)
				})

				By("Listing pod metrics for the workload through the restarted control plane", func() {
					// node metrics only pass through node filtering, but pod metrics are
					// rewritten from host pod names to tenant pod names through the name
					// mapping the restarted syncer had to rebuild, so the workload pod has
					// to show up again under its tenant name
					waitForPodMetrics(ctx, vClusterConfig, workloadNamespace, ContainElement(HaveField("Name", workloadPodName)))
				})
			})
		},
	)
}

// waitForControlPlaneReplacement waits until a control plane pod other than
// the deleted one is running with every container ready, and then requires it
// to stay that way without a single restart. Ready alone proves nothing here:
// the syncer answers its readiness probe before it runs the leader hooks, and
// the crash loop starts in those hooks, so a fresh pod looks healthy for a few
// seconds before it dies.
func waitForControlPlaneReplacement(ctx context.Context, hostClient kubernetes.Interface, vClusterNamespace, vClusterName string, deletedPodUID types.UID) {
	GinkgoHelper()

	listReplacementPods := func(g Gomega) []corev1.Pod {
		pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=vcluster,release=" + vClusterName,
		})
		g.Expect(err).To(Succeed(), "listing control plane pods in %s", vClusterNamespace)
		g.Expect(pods.Items).NotTo(BeEmpty(), "no control plane pod in %s yet", vClusterNamespace)
		for _, pod := range pods.Items {
			// the StatefulSet recreates the pod under the same name, so only the UID tells the replacement apart
			g.Expect(pod.UID).NotTo(Equal(deletedPodUID), "control plane pod %s has not been recreated yet", pod.Name)
		}
		return pods.Items
	}

	Eventually(func(g Gomega) {
		for _, pod := range listReplacementPods(g) {
			g.Expect(pod.Status.ContainerStatuses).NotTo(BeEmpty(), "pod %s has no container statuses yet", pod.Name)
			for _, container := range pod.Status.ContainerStatuses {
				if container.RestartCount > 0 {
					StopTrying(fmt.Sprintf("container %s in control plane pod %s restarted %d times after the pod was recreated, last termination: %s",
						container.Name, pod.Name, container.RestartCount, lastTerminationSummary(container))).Now()
				}
				g.Expect(container.Ready).To(BeTrue(), "container %s in pod %s is not ready yet", container.Name, pod.Name)
			}
		}
	}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())

	Consistently(func(g Gomega) {
		for _, pod := range listReplacementPods(g) {
			for _, container := range pod.Status.ContainerStatuses {
				g.Expect(container.RestartCount).To(BeZero(), "container %s in control plane pod %s restarted after the pod was recreated, last termination: %s",
					container.Name, pod.Name, lastTerminationSummary(container))
				g.Expect(container.Ready).To(BeTrue(), "container %s in pod %s stopped being ready", container.Name, pod.Name)
			}
		}
	}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
}

func lastTerminationSummary(container corev1.ContainerStatus) string {
	terminated := container.LastTerminationState.Terminated
	if terminated == nil {
		return "none recorded"
	}
	return fmt.Sprintf("exit code %d, reason %q, message %q", terminated.ExitCode, terminated.Reason, strings.TrimSpace(terminated.Message))
}

// reconnectVCluster establishes a fresh connection to the tenant cluster. The
// suite-level connection rides on a background proxy that does not survive a
// control plane pod restart.
func reconnectVCluster(ctx context.Context, vClusterName, vClusterNamespace, hostKubeconfig string) *rest.Config {
	GinkgoHelper()

	tmpFile, err := os.CreateTemp("", "vcluster-metricsproxy-kubeconfig-*")
	Expect(err).To(Succeed(), "creating temp kubeconfig file")
	Expect(tmpFile.Close()).To(Succeed())
	DeferCleanup(func(_ context.Context) {
		Expect(os.RemoveAll(tmpFile.Name())).To(Succeed(), "removing temp kubeconfig %s", tmpFile.Name())
	})

	connectCmd := connectcmd.ConnectCmd{
		CobraCmd: &cobra.Command{},
		Log:      loftlog.Discard,
		GlobalFlags: &flags.GlobalFlags{
			Namespace: vClusterNamespace,
		},
		ConnectOptions: cli.ConnectOptions{
			// the suite creates the tenant cluster with the helm driver, so do not
			// let a platform login in the local CLI config redirect the connect
			Driver:               string(config.HelmDriver),
			KubeConfig:           tmpFile.Name(),
			BackgroundProxy:      true,
			BackgroundProxyImage: constants.GetVClusterImage(),
		},
	}
	// connectCmd locates the host cluster through the ambient KUBECONFIG, which
	// parallel workers share and kind rewrites, so point it at the host
	// cluster's own kubeconfig for the duration of the call
	withKubeconfig(hostKubeconfig, func() {
		Expect(connectCmd.Run(ctx, []string{vClusterName})).To(Succeed(), "vcluster connect after the control plane restart")
	})

	var vClusterConfig *rest.Config
	Eventually(func(g Gomega) {
		data, err := os.ReadFile(tmpFile.Name())
		g.Expect(err).To(Succeed(), "reading the kubeconfig written by vcluster connect")
		g.Expect(data).NotTo(BeEmpty(), "vcluster connect has not written the kubeconfig yet")

		cfg, err := clientcmd.RESTConfigFromKubeConfig(data)
		g.Expect(err).To(Succeed(), "parsing the kubeconfig written by vcluster connect")
		client, err := kubernetes.NewForConfig(cfg)
		g.Expect(err).To(Succeed(), "building a client from the new kubeconfig")

		_, err = client.CoreV1().ServiceAccounts("default").Get(ctx, "default", metav1.GetOptions{})
		g.Expect(err).To(Succeed(), "tenant cluster not reachable through the new connection yet")
		vClusterConfig = cfg
	}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

	return vClusterConfig
}

func withKubeconfig(kubeconfig string, fn func()) {
	GinkgoHelper()
	previous, had := os.LookupEnv("KUBECONFIG")
	Expect(os.Setenv("KUBECONFIG", kubeconfig)).To(Succeed())
	defer func() {
		if had {
			Expect(os.Setenv("KUBECONFIG", previous)).To(Succeed())
		} else {
			Expect(os.Unsetenv("KUBECONFIG")).To(Succeed())
		}
	}()
	fn()
}
