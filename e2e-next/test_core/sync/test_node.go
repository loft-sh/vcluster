package test_core

import (
	"context"
	"os"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var _ = Describe("Node sync",
	Ordered,
	// labels.PR,
	labels.Core,
	labels.Sync,
	cluster.Use(clusters.NodesVCluster),
	cluster.Use(clusters.HostCluster),
	func() {
		var (
			hostClient     kubernetes.Interface
			vClusterClient kubernetes.Interface
		)

		BeforeAll(func(ctx context.Context) {
			hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
			Expect(hostClient).NotTo(BeNil())
			vClusterClient = cluster.CurrentKubeClientFrom(ctx)
			Expect(vClusterClient).NotTo(BeNil())
		})

		It("Sync nodes using label selector", func(ctx context.Context) {

			hostname := constants.GetHostClusterName() + "-control-plane"
			if kindName, ok := os.LookupEnv("KIND_NAME"); ok && kindName != "" {
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
				g.Expect(virtualNodes.Items).To(HaveLen(1), "Expected exactly one synced node")
				g.Expect(virtualNodes.Items[0].Name).To(Equal(hostSyncedNodeName), "Synced node name should match")
			}).
				WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeout).
				Should(Succeed(), "Node sync should work correctly")
		})
	})
