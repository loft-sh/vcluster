package node

import (
	"reflect"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Node sync", func() {
	f := framework.DefaultFramework
	ginkgo.It("sync all nodes", func() {
		hostNodes, err := f.HostClient.CoreV1().Nodes().List(f.Context, metav1.ListOptions{})
		framework.ExpectNoError(err)

		virtualNodes, err := f.VClusterClient.CoreV1().Nodes().List(f.Context, metav1.ListOptions{})
		framework.ExpectNoError(err)
		framework.ExpectEqual(len(hostNodes.Items), len(virtualNodes.Items))

		hostNodeLabels := make(map[string]map[string]string)
		for _, node := range hostNodes.Items {
			hostNodeLabels[node.Name] = node.Labels
		}

		virtualNodeLabels := make(map[string]map[string]string)
		for _, node := range virtualNodes.Items {
			virtualNodeLabels[node.Name] = node.Labels
		}

		framework.ExpectEqual(true, reflect.DeepEqual(hostNodeLabels, virtualNodeLabels))
	})

	ginkgo.It("fake nodes have fake kubelet service IPs", func() {
		virtualNodes, err := f.VClusterClient.CoreV1().Nodes().List(f.Context, metav1.ListOptions{})
		framework.ExpectNoError(err)

		for _, nodes := range virtualNodes.Items {
			var foundInternalIPAddress bool
			for _, address := range nodes.Status.Addresses {
				if address.Type == corev1.NodeInternalIP {
					foundInternalIPAddress = true
				}
			}

			framework.ExpectEqual(foundInternalIPAddress, true)
		}
	})
})
