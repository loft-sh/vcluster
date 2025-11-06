package test_core

import (
	"context"
	"os"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	vcluster "github.com/loft-sh/vcluster/e2e-next/setup"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var _ = Describe("Node sync",
	Ordered,
	labels.PR,
	labels.Core,
	labels.Sync,
	func() {
		var (
			hostClient     kubernetes.Interface
			vClusterName   = "nodes-test-vcluster"
			vClusterClient kubernetes.Interface
			vclusterValues string = constants.DefaultVClusterYAML
		)

		BeforeAll(func(ctx context.Context) context.Context {
			// Get host cluster client
			hostClient = cluster.CurrentKubeClientFrom(ctx)
			Expect(hostClient).NotTo(BeNil(), "Host client should not be nil")

			var err error

			ctx, err = vcluster.Create(
				vcluster.WithName(vClusterName),
				vcluster.WithValuesYAML(vclusterValues),
			)(ctx)
			Expect(err).NotTo(HaveOccurred())
			vClusterClient = vcluster.GetKubeClientFrom(ctx)
			Expect(vClusterClient).NotTo(BeNil(), "VCluster client should not be nil")
			return ctx
		})

		It("Sync nodes using label selector", func(ctx context.Context) {
			err := vcluster.WaitForControlPlane(ctx)
			Expect(err).NotTo(HaveOccurred())

			hostname := "kind-cluster-control-plane"
			if kindName, ok := os.LookupEnv("KIND_NAME"); ok {
				hostname = kindName + "-control-plane"
			}
			Eventually(func(g Gomega) {
				hostNodes, err := hostClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
				g.Expect(err).NotTo(HaveOccurred(), "Failed to list host nodes")

				virtualNodes, err := vClusterClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
				g.Expect(err).NotTo(HaveOccurred(), "Failed to list virtual nodes")
				g.Expect(virtualNodes.Items).ToNot(BeEmpty(), "Virtual nodes list should not be empty")

				hostSyncedNodeName := ""
				for _, node := range hostNodes.Items {
					if node.Labels["kubernetes.io/hostname"] == hostname {
						hostSyncedNodeName = node.Name
						break
					}
				}

				g.Expect(hostSyncedNodeName).ToNot(BeEmpty(), "Should find host node with matching hostname")
				g.Expect(hostSyncedNodeName).To(Equal(virtualNodes.Items[0].Name), "Synced node name should match")
			}).
				WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeout).
				Should(Succeed(), "Node sync should work correctly")
		})

		AfterAll(func(ctx context.Context) {
			By("Removing vcluster")
			_ = vcluster.Destroy(vClusterName)
		})
	})
