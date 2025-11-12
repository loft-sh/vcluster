package test_install

import (
	"context"
	_ "embed"

	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	vcluster "github.com/loft-sh/vcluster/e2e-next/setup"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	TestManifestName      = "test-configmap"
	TestManifestName2     = "test-configmap-2"
	TestManifestNamespace = "default"
)

var (
	//go:embed vcluster-init-manifest.yaml
	vclusterInitManifestValues string
)

var _ = Describe("Init manifests are synced and applied as expected",
	Ordered,
	labels.Deploy,

	func() {
		var (
			vClusterName   = "init-manifests-test-vcluster"
			vclusterClient kubernetes.Interface
		)

		BeforeAll(func(ctx context.Context) context.Context {

			var err error

			By("Create vCluster")
			ctx, err = vcluster.Create(
				vcluster.WithName(vClusterName),
				vcluster.WithValuesYAML(vclusterInitManifestValues),
			)(ctx)
			Expect(err).NotTo(HaveOccurred())
			By("Wait for vCluster control plane")
			err = vcluster.WaitForControlPlane(ctx)
			Expect(err).NotTo(HaveOccurred())
			vclusterClient = vcluster.GetKubeClientFrom(ctx)
			Expect(vclusterClient).NotTo(BeNil(), "VCluster client should not be nil")
			return ctx
		})

		It("Test if manifest is synced with the vcluster", func(ctx context.Context) {

			Eventually(func(g Gomega) {
				manifest, err := vclusterClient.CoreV1().ConfigMaps(TestManifestNamespace).Get(ctx, TestManifestName, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred(), "ConfigMap should exist")
				g.Expect(manifest.Data).To(HaveKey("foo"), "ConfigMap should have the foo key")
				g.Expect(manifest.Data["foo"]).To(Equal("bar"), "ConfigMap foo value should be 'bar'")
			}).
				WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeout).
				Should(Succeed(), "Manifest should be synced")
		})

		It("Test if manifest template is synced with the vcluster", func(ctx context.Context) {
			Eventually(func(g Gomega) {
				manifest, err := vclusterClient.CoreV1().ConfigMaps(TestManifestNamespace).Get(ctx, TestManifestName2, metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred(), "ConfigMap should exist")
				g.Expect(manifest.Data).To(HaveKey("foo"), "ConfigMap should have the foo key")
				g.Expect(manifest.Data["foo"]).To(Equal(vClusterName), "ConfigMap foo value should equal vcluster name")
			}).
				WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeout).
				Should(Succeed(), "Manifest template should be synced")
		})
		AfterAll(func(ctx context.Context) {
			By("Removing vCluster")
			_ = vcluster.Destroy(vClusterName)
		})
	},
)
