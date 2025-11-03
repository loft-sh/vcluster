package e2e_next

import (
	"context"
	"fmt"
	"reflect"

	"github.com/loft-sh/e2e-framework/pkg/e2e"
	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	vcluster "github.com/loft-sh/vcluster/e2e-next/setup"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var _ = e2e.Describe("map default/kubernetes endpoint to physical vcluster endpoint",
	labels.Core,
	labels.PR,
	func() {
		var (
			hostClient        kubernetes.Interface
			vClusterName      = "k8s-default-endpoint-test-vcluster"
			vClusterNamespace string
		)

		e2e.BeforeAll(func(ctx context.Context) context.Context {
			// Get host cluster client
			hostClient = cluster.CurrentKubeClientFrom(ctx)
			vClusterNamespace = "vcluster-" + vClusterName

			// Create vcluster values inline
			vclusterValues := `controlPlane:
  statefulSet:
    image:
      registry: ""
      repository: vcluster
      tag: e2e-latest
`
			var err error

			ctx, err = vcluster.Create(
				vcluster.WithName(vClusterName),
				vcluster.WithValuesYAML(vclusterValues),
			)(ctx)
			Expect(err).NotTo(HaveOccurred())
			err = vcluster.WaitForControlPlane(ctx)
			Expect(err).NotTo(HaveOccurred())
			return ctx
		})

		e2e.It("Test default/kubernetes endpoints matches with vcluster service endpoint", func(ctx context.Context) {
			kubeClient := vcluster.GetKubeClientFrom(ctx)
			Eventually(func(g Gomega) {
				hostClusterEndpoint, err := hostClient.CoreV1().Endpoints(vClusterNamespace).Get(ctx, vClusterName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get host cluster endpoint")

				vclusterEndpoint, err := kubeClient.CoreV1().Endpoints("default").Get(ctx, "kubernetes", metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get vcluster endpoint")

				hostClusterIps := make([]string, 0)
				hostClusterPorts := make([]int32, 0)
				vClusterIps := make([]string, 0)
				vClusterPorts := make([]int32, 0)

				g.Expect(len(hostClusterEndpoint.Subsets)).To(BeNumerically(">", 0), "Host cluster endpoint should have at least one subset")
				g.Expect(len(vclusterEndpoint.Subsets)).To(BeNumerically(">", 0), "VCluster endpoint should have at least one subset")

				for _, address := range hostClusterEndpoint.Subsets[0].Addresses {
					hostClusterIps = append(hostClusterIps, address.IP)
				}

				for _, port := range hostClusterEndpoint.Subsets[0].Ports {
					if port.Name == "kubelet" {
						continue
					}
					hostClusterPorts = append(hostClusterPorts, port.Port)
				}

				for _, address := range vclusterEndpoint.Subsets[0].Addresses {
					vClusterIps = append(vClusterIps, address.IP)
				}

				for _, port := range vclusterEndpoint.Subsets[0].Ports {
					vClusterPorts = append(vClusterPorts, port.Port)
				}

				// Add detailed error output if they don't match
				if !reflect.DeepEqual(hostClusterIps, vClusterIps) || !reflect.DeepEqual(hostClusterPorts, vClusterPorts) {
					fmt.Printf("IPs mismatch - Host: %v, VCluster: %v\n", hostClusterIps, vClusterIps)
					fmt.Printf("Ports mismatch - Host: %v, VCluster: %v\n", hostClusterPorts, vClusterPorts)
				}
				g.Expect(hostClusterIps).To(Equal(vClusterIps), "IPs should match between host and vcluster endpoints")
				g.Expect(hostClusterPorts).To(Equal(vClusterPorts), "Ports should match between host and vcluster endpoints")
			}).
				WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeout).
				Should(Succeed(), "Endpoints should match after waiting")
		})
	},
)
