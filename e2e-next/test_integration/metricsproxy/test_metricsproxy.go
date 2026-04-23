// Package metricsproxy contains metrics proxy integration tests.
package metricsproxy

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	apiregistrationv1clientset "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1"
	metricsv1beta1client "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
)

// MetricsProxySpec registers metrics proxy integration tests.
// The vCluster must be configured with
// integrations.metricsServer.enabled: true and the host cluster must have
// metrics-server installed (handled by MetricsProxyVCluster's preSetup).
func MetricsProxySpec() {
	Describe("Metrics proxy integration",
		labels.Integration,
		func() {
			var vClusterConfig *rest.Config

			BeforeEach(func(ctx context.Context) {
				currentClusterName := cluster.CurrentClusterNameFrom(ctx)
				vClusterConfig = cluster.From(ctx, currentClusterName).KubernetesRestConfig()
				Expect(vClusterConfig).NotTo(BeNil())
			})

			It("should register and expose the metrics API service as Available", func(ctx context.Context) {
				apiRegistrationClient := apiregistrationv1clientset.NewForConfigOrDie(vClusterConfig)

				Eventually(func(g Gomega) {
					apiService, err := apiRegistrationClient.APIServices().Get(ctx, "v1beta1.metrics.k8s.io", metav1.GetOptions{})
					g.Expect(err).To(Succeed(), "failed to get APIService v1beta1.metrics.k8s.io")
					g.Expect(apiService.Status.Conditions).To(ContainElement(SatisfyAll(
						HaveField("Type", apiregistrationv1.Available),
						HaveField("Status", apiregistrationv1.ConditionTrue),
					)), "APIService v1beta1.metrics.k8s.io not yet Available=True, conditions: %v", apiService.Status.Conditions)
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
			})

			It("should return non-empty node metrics and pod metrics from kube-system", func(ctx context.Context) {
				metricsClient := metricsv1beta1client.NewForConfigOrDie(vClusterConfig)

				By("waiting for node metrics to be available", func() {
					Eventually(func(g Gomega) {
						nodeMetricsList, err := metricsClient.NodeMetricses().List(ctx, metav1.ListOptions{})
						g.Expect(err).To(Succeed(), "failed to list node metrics")
						g.Expect(nodeMetricsList.Items).NotTo(BeEmpty(),
							"expected at least one node metrics entry, got %d", len(nodeMetricsList.Items))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("waiting for pod metrics in kube-system to be available", func() {
					Eventually(func(g Gomega) {
						podMetricsList, err := metricsClient.PodMetricses("kube-system").List(ctx, metav1.ListOptions{})
						g.Expect(err).To(Succeed(), "failed to list pod metrics in kube-system")
						g.Expect(podMetricsList.Items).NotTo(BeEmpty(),
							"expected at least one pod metrics entry in kube-system, got %d", len(podMetricsList.Items))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})
		},
	)
}
