package lifecycle

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var _ = Describe("Pause and resume vCluster",
	labels.Core,
	cluster.Use(clusters.PauseResumeVCluster),
	cluster.Use(clusters.HostCluster),
	func() {
		var (
			vClusterName      = clusters.PauseResumeVClusterName
			vClusterNamespace = "vcluster-" + vClusterName
			hostClient        kubernetes.Interface
		)

		BeforeEach(func(ctx context.Context) {
			hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
			Expect(hostClient).NotTo(BeNil())
		})

		It("pauses and resumes the vCluster via CLI", func(ctx context.Context) {
			kubeContext := "kind-" + constants.GetHostClusterName()
			vclusterBin := filepath.Join(os.Getenv("GOBIN"), "vcluster")

			By("Verifying vCluster pods are running", func() {
				Eventually(func(g Gomega) {
					pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
						LabelSelector: "app=vcluster",
					})
					g.Expect(err).NotTo(HaveOccurred(), "failed to list vcluster pods")
					g.Expect(pods.Items).NotTo(BeEmpty(), "expected at least one vcluster pod")
				}).WithPolling(constants.PollingInterval).
					WithTimeout(constants.PollingTimeoutShort).
					Should(Succeed())
			})

			By("Pausing the vCluster", func() {
				cmd := exec.CommandContext(ctx, vclusterBin, "pause", vClusterName,
					"-n", vClusterNamespace,
					"--context", kubeContext)
				output, err := cmd.CombinedOutput()
				Expect(err).NotTo(HaveOccurred(), "vcluster pause failed: %s", string(output))
			})

			By("Verifying all vCluster pods are removed after pause", func() {
				Eventually(func(g Gomega) {
					pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
						LabelSelector: "app=vcluster",
					})
					g.Expect(err).NotTo(HaveOccurred(), "failed to list vcluster pods after pause")
					g.Expect(pods.Items).To(BeEmpty(), "expected no vcluster pods after pause")
				}).WithPolling(constants.PollingInterval).
					WithTimeout(constants.PollingTimeoutLong).
					Should(Succeed())
			})

			By("Resuming the vCluster", func() {
				cmd := exec.CommandContext(ctx, vclusterBin, "resume", vClusterName,
					"-n", vClusterNamespace,
					"--context", kubeContext)
				output, err := cmd.CombinedOutput()
				Expect(err).NotTo(HaveOccurred(), "vcluster resume failed: %s", string(output))
			})

			By("Waiting for all vCluster pods to return to Running state", func() {
				Eventually(func(g Gomega) {
					pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
						LabelSelector: "app=vcluster",
					})
					g.Expect(err).NotTo(HaveOccurred(), "failed to list vcluster pods after resume")
					g.Expect(pods.Items).NotTo(BeEmpty(), "expected at least one vcluster pod after resume")
					for _, pod := range pods.Items {
						g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning),
							"pod %s is in phase %s, expected Running", pod.Name, pod.Status.Phase)
					}
				}).WithPolling(constants.PollingInterval).
					WithTimeout(constants.PollingTimeoutLong).
					Should(Succeed())
			})
		})
	},
)
