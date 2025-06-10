package node

import (
	"os"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Node sync", func() {
	f := framework.DefaultFramework
	ginkgo.It("sync nodes using label selector", func() {
		hostNodes, err := f.HostClient.CoreV1().Nodes().List(f.Context, metav1.ListOptions{})
		framework.ExpectNoError(err)

		virtualNodes, err := f.VClusterClient.CoreV1().Nodes().List(f.Context, metav1.ListOptions{})
		framework.ExpectNoError(err)

		hostname := "kind-control-plane"

		if kindName, ok := os.LookupEnv("KIND_NAME"); ok {
			hostname = kindName + "-control-plane"
		}

		hostSyncedNodeName := ""
		hostNodeLabels := make(map[string]map[string]string)
		for _, node := range hostNodes.Items {
			hostNodeLabels[node.Name] = node.Labels
			if node.Labels["kubernetes.io/hostname"] == hostname {
				hostSyncedNodeName = node.Name
				break
			}
		}

		framework.ExpectEqual(hostSyncedNodeName, virtualNodes.Items[0].Name)
	})
})
