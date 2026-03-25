package test_vind

import (
	"context"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
)

var _ = Describe("Docker driver", labels.Vind, labels.PR, func() {
	var clusterName string

	BeforeEach(func() {
		clusterName = "e2e-vind-" + random.String(6)
	})

	It("should create, connect, and delete a vcluster", func(ctx context.Context) {
		var (
			cpContainer  string
			network      string
			volumePrefix string
		)

		By("Creating vcluster with docker driver", func() {
			_, err := runVCluster(ctx, "create", clusterName, "--driver", "docker", "--connect=false", "--chart-version", getChartVersion())
			Expect(err).To(Succeed())
		})

		By("Verifying control plane container is running", func() {
			cpContainer = controlPlaneContainerName(clusterName)
			Expect(dockerContainerRunning(ctx, cpContainer)).To(BeTrue(), "control plane container should be running")
		})

		By("Verifying Docker network exists", func() {
			network = networkName(clusterName)
			Expect(dockerNetworkExists(ctx, network)).To(BeTrue(), "docker network should exist")
		})

		By("Verifying control plane volumes exist", func() {
			volumePrefix = controlPlaneVolumePrefix(clusterName)
			Expect(dockerVolumesExist(ctx, volumePrefix)).To(BeTrue(), "control plane volumes should exist")
		})

		By("Connecting to vcluster and verifying at least one node is ready", func() {
			Eventually(func(g Gomega) {
				kubeConfig, err := runVCluster(ctx, "connect", clusterName, "--driver", "docker", "--print")
				g.Expect(err).To(Succeed(), "failed to get kubeconfig")
				client, err := kubeClientFromKubeConfig([]byte(kubeConfig))
				g.Expect(err).To(Succeed(), "failed to create kube client")
				nodeList, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
				g.Expect(err).To(Succeed(), "list nodes failed")
				readyCount := 0
				for _, node := range nodeList.Items {
					for _, cond := range node.Status.Conditions {
						if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
							readyCount++
						}
					}
				}
				g.Expect(readyCount).To(BeNumerically(">=", 1), "expected at least 1 ready node")
			}).
				WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeoutLong).
				Should(Succeed())
		})

		By("Deleting vcluster", func() {
			_, err := runVCluster(ctx, "delete", clusterName, "--driver", "docker")
			Expect(err).To(Succeed())
		})

		By("Verifying control plane container is gone", func() {
			Expect(dockerContainerExists(ctx, cpContainer)).To(BeFalse(), "control plane container should be removed")
		})

		By("Verifying Docker network is gone", func() {
			Expect(dockerNetworkExists(ctx, network)).To(BeFalse(), "docker network should be removed")
		})

		By("Verifying volumes are gone", func() {
			Expect(dockerVolumesExist(ctx, volumePrefix)).To(BeFalse(), "control plane volumes should be removed")
		})

		By("Verifying no containers with cluster name prefix remain", func() {
			out, err := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "name=^vcluster.cp."+clusterName+"$", "--format", "{{.ID}}").Output()
			Expect(err).To(Succeed())
			Expect(strings.TrimSpace(string(out))).To(BeEmpty(), "no control plane containers should remain")

			out, err = exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "name=^vcluster.node."+clusterName+"\\.", "--format", "{{.ID}}").Output()
			Expect(err).To(Succeed())
			Expect(strings.TrimSpace(string(out))).To(BeEmpty(), "no worker node containers should remain")

			out, err = exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "name=^vcluster.lb."+clusterName+"\\.", "--format", "{{.ID}}").Output()
			Expect(err).To(Succeed())
			Expect(strings.TrimSpace(string(out))).To(BeEmpty(), "no load balancer containers should remain")
		})
	})
})
