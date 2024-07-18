package k8sdefaultendpoint

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
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

		waitErr := wait.PollUntilContextTimeout(ctx, time.Millisecond*500, framework.PollTimeout*2, true, func(ctx context.Context) (done bool, err error) {
			hostClusterEndpoint, err := f.HostClient.CoreV1().Endpoints(f.VclusterNamespace).Get(ctx, "vcluster", v1.GetOptions{})
			if err != nil {
				return false, err
			}

			vclusterEndpoint, err := f.VClusterClient.CoreV1().Endpoints("default").Get(ctx, "kubernetes", v1.GetOptions{})
			if err != nil {
				return false, err
			}

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

			if !reflect.DeepEqual(hostClusterIps, vClusterIps) || !reflect.DeepEqual(hostClusterPorts, vClusterPorts) {
				out, _ := yaml.Marshal(vclusterEndpoint)

				fmt.Println("IPs", hostClusterIps, vClusterIps, string(out))
				fmt.Println("Ports", hostClusterPorts, vClusterPorts)
				return false, nil
			}

			return true, nil
		})
		framework.ExpectNoError(waitErr, "error after waiting")
	})
})
