package k8sdefaultendpoint

import (
	"reflect"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("map default/kubernetes endpoint to physical vcluster endpoint", func() {
	var (
		f *framework.Framework
	)

	ginkgo.JustBeforeEach(func() {
		// use default framework
		f = framework.DefaultFramework
	})

	ginkgo.It("Test default/kubernetes endpoints matches with vcluster service endpoint", func() {
		ctx := f.Context

		hostClusterEndpoint, err := f.HostClient.CoreV1().Endpoints(f.VclusterNamespace).Get(ctx, "vcluster", v1.GetOptions{})
		framework.ExpectNoError(err)

		vclusterEndpoint, err := f.VclusterClient.CoreV1().Endpoints("default").Get(ctx, "kubernetes", v1.GetOptions{})
		framework.ExpectNoError(err)

		hostClusterIps := make([]string, 0)
		hostClusterPorts := make([]int32, 0)
		vClusterIps := make([]string, 0)
		vClusterPorts := make([]int32, 0)

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

		ok := reflect.DeepEqual(hostClusterIps, vClusterIps)
		framework.ExpectEqual(ok, true)
		ok = reflect.DeepEqual(hostClusterPorts, vClusterPorts)
		framework.ExpectEqual(ok, true)

	})
})
