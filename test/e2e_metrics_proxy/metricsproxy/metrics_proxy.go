package metricsproxy

import (
	"context"
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	apiregistrationv1clientset "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1"

	metricsv1beta1client "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
)

var _ = ginkgo.Describe("Target Namespace", func() {
	f := framework.DefaultFramework

	ginkgo.It("Make sure the metrics api service is registered and available", func() {
		err := wait.PollUntilContextTimeout(f.Context, time.Second, time.Minute*2, false, func(ctx context.Context) (bool, error) {
			apiRegistrationClient := apiregistrationv1clientset.NewForConfigOrDie(f.VClusterConfig)
			apiService, err := apiRegistrationClient.APIServices().Get(ctx, "v1beta1.metrics.k8s.io", metav1.GetOptions{})
			if err != nil {
				return false, nil
			}

			if apiService.Status.Conditions[0].Type != apiregistrationv1.Available {
				return false, nil
			}

			return true, nil
		})
		framework.ExpectNoError(err)
	})

	ginkgo.It("Make sure get nodeMetrics and podMetrics succeed", func() {
		err := wait.PollUntilContextTimeout(f.Context, time.Second, time.Minute*2, false, func(ctx context.Context) (bool, error) {
			metricsClient := metricsv1beta1client.NewForConfigOrDie(f.VClusterConfig)

			nodeMetricsList, err := metricsClient.NodeMetricses().List(ctx, metav1.ListOptions{})
			if err != nil {
				fmt.Fprintf(ginkgo.GinkgoWriter, "error getting node metrics list %v", err)
				return false, nil
			}

			if len(nodeMetricsList.Items) == 0 {
				fmt.Fprintf(ginkgo.GinkgoWriter, "expecting node metrics list to have at least 1 entry, got %d", len(nodeMetricsList.Items))
				return false, nil
			}

			podMetricsList, err := metricsClient.PodMetricses("kube-system").List(ctx, metav1.ListOptions{})
			if err != nil {
				fmt.Fprintf(ginkgo.GinkgoWriter, "error getting pod metrics list %v", err)
				return false, nil
			}

			if len(podMetricsList.Items) == 0 {
				fmt.Fprintf(ginkgo.GinkgoWriter, "expecting pod metrics list to have at least 1 entry, got %d", len(podMetricsList.Items))
				return false, nil
			}

			return true, nil
		})

		framework.ExpectNoError(err)
	})
})
