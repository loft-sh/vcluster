// Package nodesync contains all-nodes sync mode tests.
package nodesync

import (
	"context"
	"reflect"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NodeSyncSpec registers node sync tests.
// The vCluster must be configured with sync.fromHost.nodes.selector.all=true
// so that all host nodes are synced into the virtual cluster.
func NodeSyncSpec() {
	Describe("Node sync",
		labels.Sync,
		func() {
			var (
				hostClient     kubernetes.Interface
				vClusterClient kubernetes.Interface
				vClusterName   string
			)

			BeforeEach(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
			})

			It("should sync all host nodes into the virtual cluster with matching labels", func(ctx context.Context) {
				Eventually(func(g Gomega) {
					hostNodes, err := hostClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
					g.Expect(err).To(Succeed(), "failed to list host nodes")

					virtualNodes, err := vClusterClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
					g.Expect(err).To(Succeed(), "failed to list virtual nodes")

					g.Expect(virtualNodes.Items).To(HaveLen(len(hostNodes.Items)),
						"expected %d virtual nodes to match %d host nodes", len(hostNodes.Items), len(virtualNodes.Items))

					hostNodeLabels := make(map[string]map[string]string)
					for _, node := range hostNodes.Items {
						hostNodeLabels[node.Name] = node.Labels
					}

					virtualNodeLabels := make(map[string]map[string]string)
					for _, node := range virtualNodes.Items {
						// The marker label is only present on virtual nodes; exclude it
						// before comparing so host and virtual label sets match.
						nodeCopy := node.DeepCopy()
						if nodeCopy.Labels[translate.MarkerLabel] == vClusterName {
							delete(nodeCopy.Labels, translate.MarkerLabel)
						}
						virtualNodeLabels[nodeCopy.Name] = nodeCopy.Labels
					}

					g.Expect(reflect.DeepEqual(hostNodeLabels, virtualNodeLabels)).To(BeTrue(),
						"virtual node labels do not match host node labels: host=%v, virtual=%v",
						hostNodeLabels, virtualNodeLabels)
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
			})

			It("should provide an InternalIP address for every synced node", func(ctx context.Context) {
				Eventually(func(g Gomega) {
					virtualNodes, err := vClusterClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
					g.Expect(err).To(Succeed(), "failed to list virtual nodes")
					g.Expect(virtualNodes.Items).NotTo(BeEmpty(), "virtual nodes list should not be empty")

					for _, node := range virtualNodes.Items {
						foundInternalIP := false
						for _, address := range node.Status.Addresses {
							if address.Type == corev1.NodeInternalIP {
								foundInternalIP = true
								break
							}
						}
						g.Expect(foundInternalIP).To(BeTrue(),
							"node %s has no InternalIP address in Status.Addresses", node.Name)
					}
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
			})
		},
	)
}
