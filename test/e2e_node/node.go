package e2enode

import (
	"reflect"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Node sync", func() {
	f := framework.DefaultFramework
	ginkgo.It("sync all nodes", func() {
		hostNodes, err := f.HostClient.CoreV1().Nodes().List(f.Context, metav1.ListOptions{})
		framework.ExpectNoError(err)

		virtualNodes, err := f.VclusterClient.CoreV1().Nodes().List(f.Context, metav1.ListOptions{})
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
})
