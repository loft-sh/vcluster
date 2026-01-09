package test_deploy

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	e2eLabels "github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/controllers/deploy"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

const (
	ChartName            = "ingress-nginx"
	ChartNamespace       = "ingress-nginx"
	ChartOCIName         = "fluent-bit"
	ChartOCIInstanceName = "fluent-bit"
	ChartOCINamespace    = "fluent-bit"
)

var _ = Describe("Helm charts (regular and OCI) are synced and applied as expected",
	Ordered,
	e2eLabels.Deploy,
	cluster.Use(clusters.HelmChartsVCluster),
	func() {
		var (
			HelmSecretLabels = map[string]string{
				"owner": "helm",
				"name":  ChartName,
			}
			HelmOCIDeploymentLabels = map[string]string{
				"app.kubernetes.io/instance": ChartOCIInstanceName,
				"app.kubernetes.io/name":     ChartOCIName,
			}
			vClusterClient kubernetes.Interface
		)

		BeforeAll(func(ctx context.Context) {
			vClusterClient = cluster.CurrentKubeClientFrom(ctx)
			Expect(vClusterClient).NotTo(BeNil())
		})

		It("Test if configmap for both charts gets applied", func(ctx context.Context) {
			Eventually(func(g Gomega) {
				cm, err := vClusterClient.CoreV1().ConfigMaps(deploy.VClusterDeployConfigMapNamespace).
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
				WithTimeout(constants.PollingTimeout).
				Should(Succeed(), "Both charts should be successfully deployed")
		})

		It("Test nginx release secret existence in vcluster (regular chart)", func(ctx context.Context) {
			Eventually(func(g Gomega) {
				secList, err := vClusterClient.CoreV1().Secrets(ChartNamespace).List(ctx, metav1.ListOptions{
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

		It("Test fluent-bit release deployment existence in vcluster (OCI chart)", func(ctx context.Context) {
			Eventually(func(g Gomega) []appsv1.Deployment {
				deployList, err := vClusterClient.AppsV1().Deployments(ChartOCINamespace).List(ctx, metav1.ListOptions{
					LabelSelector: labels.SelectorFromSet(HelmOCIDeploymentLabels).String(),
				})
				g.Expect(err).NotTo(HaveOccurred(), "Should be able to list deployments")
				return deployList.Items
			}).
				WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeout).
				Should(HaveLen(1), "Should have exactly one fluent-bit deployment")
		})
	})
