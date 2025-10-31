package e2e_next

import (
	"context"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/e2e"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	e2eLabels "github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/e2e-next/setup"

	"github.com/loft-sh/vcluster/pkg/controllers/deploy"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
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

var _ = e2e.Describe("Helm charts (regular and OCI) are synced and applied as expected",
	e2eLabels.Test,
	e2eLabels.PR,
	func() {
		var (
			vcluster     *setup.VClusterSetup
			vClusterName = "helm-charts-test-vcluster"

			HelmSecretLabels = map[string]string{
				"owner": "helm",
				"name":  ChartName,
			}
			HelmOCIDeploymentLabels = map[string]string{
				"app.kubernetes.io/instance": ChartOCIInstanceName,
				"app.kubernetes.io/name":     ChartOCIName,
			}
		)

		e2e.BeforeAll(func(ctx context.Context) context.Context {
			vclusterValues := `controlPlane:
  statefulSet:
    image:
      registry: ""
      repository: vcluster
      tag: e2e-latest
experimental:
  deploy:
    vcluster:
      helm:
        - chart:
            name: ingress-nginx
            repo: https://kubernetes.github.io/ingress-nginx
            version: 4.1.1
          release:
            name: ingress-nginx
            namespace: ingress-nginx
          timeout: "50s"
        - chart:
            name: fluent-bit
            repo: oci://registry-1.docker.io/bitnamicharts
            version: 0.4.3
          release:
            name: fluent-bit
            namespace: fluent-bit
          timeout: "50s"
`
			vcluster = setup.CreateVClusterFromValues(ctx, vClusterName, vclusterValues)
			vcluster.WaitForControlPlane(ctx)
			return ctx
		})

		e2e.It("Test if configmap for both charts gets applied", func(ctx context.Context) {
			Eventually(func(g Gomega) {
				cm, err := vcluster.KubeClient.CoreV1().ConfigMaps(deploy.VClusterDeployConfigMapNamespace).
					Get(ctx, deploy.VClusterDeployConfigMap, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred(), "Deploy configmap should exist")
				status := deploy.ParseStatus(cm)
				g.Expect(status.Charts).To(HaveLen(2), "Should have 2 charts configured")
				for _, chart := range status.Charts {
					g.Expect(chart.Phase).To(
						Equal(string(deploy.StatusSuccess)),
						fmt.Sprintf("Chart %s is not in Success phase, got phase=%s, reason=%s, message=%s", chart.Name, chart.Phase, chart.Reason, chart.Message))
				}
			}).
				WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeoutShort).
				Should(Succeed(), "Both charts should be successfully deployed")
		})

		e2e.It("Test nginx release secret existence in vcluster (regular chart)", func(ctx context.Context) {
			Eventually(func(g Gomega) {
				secList, err := vcluster.KubeClient.CoreV1().Secrets(ChartNamespace).List(ctx, metav1.ListOptions{
					LabelSelector: labels.SelectorFromSet(HelmSecretLabels).String(),
				})
				g.Expect(err).NotTo(HaveOccurred(), "Should be able to list secrets")
				g.Expect(secList.Items).To(HaveLen(1), "Should have exactly one helm secret")
				g.Expect(secList.Items[0].Data).NotTo(BeEmpty(), "Secret data should not be empty")
				g.Expect(secList.Items[0].Data["release"]).NotTo(BeEmpty(), "Release data should not be empty")
			}).
				WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeout).
				Should(Succeed(), "Nginx helm release secret should exist")
		})

		e2e.It("Test fluent-bit release deployment existence in vcluster (OCI chart)", func(ctx context.Context) {
			Eventually(func(g Gomega) []appsv1.Deployment {
				deployList, err := vcluster.KubeClient.AppsV1().Deployments(ChartOCINamespace).List(ctx, metav1.ListOptions{
					LabelSelector: labels.SelectorFromSet(HelmOCIDeploymentLabels).String(),
				})
				g.Expect(err).NotTo(HaveOccurred(), "Should be able to list deployments")
				return deployList.Items
			}).
				WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeout).
				Should(HaveLen(1), "Should have exactly one fluent-bit deployment")
		})
	},
)
