package e2emetricsproxy

import (
	"context"
	"time"

	"github.com/loft-sh/vcluster/pkg/metricsapiservice"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	apiregistrationv1clientset "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1"
)

var _ = ginkgo.Describe("Target Namespace", func() {
	f := framework.DefaultFramework

	ginkgo.It("Make sure the metrics api service is registered and available", func() {
		err := wait.PollUntilContextTimeout(f.Context, time.Second, time.Minute*1, false, func(ctx context.Context) (done bool, err error) {
			apiRegistrationClient := apiregistrationv1clientset.NewForConfigOrDie(f.VclusterConfig)
			apiService, err := apiRegistrationClient.APIServices().Get(f.Context, metricsapiservice.MetricsAPIServiceName, metav1.GetOptions{})
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
})
