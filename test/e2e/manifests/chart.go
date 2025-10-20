package manifests

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/controllers/deploy"
	"github.com/loft-sh/vcluster/test/framework"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	ChartName            = "ingress-nginx"
	ChartNamespace       = "ingress-nginx"
	ChartOCIName         = "fluent-bit"
	ChartOCIInstanceName = "fluent-bit"
	ChartOCINamespace    = "fluent-bit"
)

var _ = Describe("Helm charts (regular and OCI) are synced and applied as expected", func() {
	var (
		f                = framework.DefaultFramework
		HelmSecretLabels = map[string]string{
			"owner": "helm",
			"name":  ChartName,
		}
		HelmOCIDeploymentLabels = map[string]string{
			"app.kubernetes.io/instance": ChartOCIInstanceName,
			"app.kubernetes.io/name":     ChartOCIName,
		}
	)

	It("Test if configmap for both charts gets applied", func(ctx context.Context) {
		Eventually(func(g Gomega, ctx context.Context) {
			cm, err := f.VClusterClient.CoreV1().ConfigMaps(deploy.VClusterDeployConfigMapNamespace).
				Get(ctx, deploy.VClusterDeployConfigMap, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			status := deploy.ParseStatus(cm)
			g.Expect(status.Charts).To(HaveLen(2))
			for _, chart := range status.Charts {
				g.Expect(chart.Phase).To(
					Equal(string(deploy.StatusSuccess)),
					fmt.Sprintf("Chart %s is not in Success phase, got phase=%s, reason=%s, message=%s", chart.Name, chart.Phase, chart.Reason, chart.Message))
			}
		}).WithContext(ctx).
			WithPolling(framework.PollInterval).
			WithTimeout(framework.PollTimeoutLong).
			Should(Succeed())
	})

	It("Test nginx release secret existence in vcluster (regular chart)", func(ctx context.Context) {
		Eventually(func(g Gomega, ctx context.Context) {
			secList, err := f.VClusterClient.CoreV1().Secrets(ChartNamespace).List(ctx, metav1.ListOptions{
				LabelSelector: labels.SelectorFromSet(HelmSecretLabels).String(),
			})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(secList.Items).To(HaveLen(1))
			g.Expect(secList.Items[0].Data).NotTo(BeEmpty())
			g.Expect(secList.Items[0].Data["release"]).NotTo(BeEmpty())
		}).WithContext(ctx).
			WithPolling(framework.PollInterval).
			WithTimeout(framework.PollTimeout).
			Should(Succeed())
	})

	It("Test fluent-bit release deployment existence in vcluster (OCI chart)", func(ctx context.Context) {
		Eventually(func(g Gomega, ctx context.Context) int {
			deployList, err := f.VClusterClient.AppsV1().Deployments(ChartOCINamespace).List(ctx, metav1.ListOptions{
				LabelSelector: labels.SelectorFromSet(HelmOCIDeploymentLabels).String(),
			})
			g.Expect(err).NotTo(HaveOccurred())
			return len(deployList.Items)
		}).WithContext(ctx).
			WithPolling(framework.PollInterval).
			WithTimeout(framework.PollTimeout).
			Should(HaveLen(1))
	})
})
