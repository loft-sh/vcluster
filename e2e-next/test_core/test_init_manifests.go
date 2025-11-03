package test_core

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/e2e"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	vcluster "github.com/loft-sh/vcluster/e2e-next/setup"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TestManifestName      = "test-configmap"
	TestManifestName2     = "test-configmap-2"
	TestManifestNamespace = "default"
)

var _ = e2e.Describe("Init manifests are synced and applied as expected",
	labels.Core,
	labels.PR,
	func() {
		var (
			vClusterName = "init-manifests-test-vcluster"
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
      manifests: |-
        apiVersion: v1
        kind: ConfigMap
        metadata:
          name: test-configmap
        data:
          foo: bar
      manifestsTemplate: |-
        apiVersion: v1
        kind: ConfigMap
        metadata:
          name: test-configmap-2
        data:
          foo: {{ .Release.Name }}
`

			var err error

			ctx, err = vcluster.Create(
				vcluster.WithName(vClusterName),
				vcluster.WithValuesYAML(vclusterValues),
			)(ctx)
			Expect(err).NotTo(HaveOccurred())
			err = vcluster.WaitForControlPlane(ctx)
			Expect(err).NotTo(HaveOccurred())
			return ctx
		})

		e2e.It("Test if manifest is synced with the vcluster", func(ctx context.Context) {
			kubeClient := vcluster.GetKubeClientFrom(ctx)
			Eventually(func(g Gomega) {
				manifest, err := kubeClient.CoreV1().ConfigMaps(TestManifestNamespace).Get(ctx, TestManifestName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred(), "ConfigMap should exist")
				g.Expect(manifest.Data).To(HaveKey("foo"), "ConfigMap should have the foo key")
				g.Expect(manifest.Data["foo"]).To(Equal("bar"), "ConfigMap foo value should be 'bar'")
			}).
				WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeout).
				Should(Succeed(), "Manifest should be synced")
		})

		e2e.It("Test if manifest template is synced with the vcluster", func(ctx context.Context) {
			kubeClient := vcluster.GetKubeClientFrom(ctx)
			Eventually(func(g Gomega) {
				manifest, err := kubeClient.CoreV1().ConfigMaps(TestManifestNamespace).Get(ctx, TestManifestName2, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred(), "ConfigMap should exist")
				g.Expect(manifest.Data).To(HaveKey("foo"), "ConfigMap should have the foo key")
				g.Expect(manifest.Data["foo"]).To(Equal(vClusterName), "ConfigMap foo value should equal vcluster name")
			}).
				WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeout).
				Should(Succeed(), "Manifest template should be synced")
		})
		e2e.AfterAll(func(ctx context.Context) {
			By("Removing vcluster")
			_ = vcluster.Destroy(vClusterName)
		})
	},
)
