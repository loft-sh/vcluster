package test_deploy

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	testManifestName      = "test-configmap"
	testManifestName2     = "test-configmap-2"
	testManifestNamespace = "default"
)

// DescribeInitManifests registers init manifest deployment tests against the given vCluster.
func DescribeInitManifests(vcluster suite.Dependency) bool {
	return Describe("Init manifests are synced and applied as expected",
		labels.Deploy,
		labels.PR,
		cluster.Use(vcluster),
		func() {
			var (
				vClusterName   string
				vClusterClient kubernetes.Interface
			)

			BeforeEach(func(ctx context.Context) {
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
			})

			It("Test if manifest is synced with the vcluster", func(ctx context.Context) {
				Eventually(func(g Gomega) {
					manifest, err := vClusterClient.CoreV1().ConfigMaps(testManifestNamespace).Get(ctx, testManifestName, metav1.GetOptions{})
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
					manifest, err := vClusterClient.CoreV1().ConfigMaps(testManifestNamespace).Get(ctx, testManifestName2, metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred(), "ConfigMap should exist")
					g.Expect(manifest.Data).To(HaveKey("foo"), "ConfigMap should have the foo key")
					g.Expect(manifest.Data["foo"]).To(Equal(vClusterName), "ConfigMap foo value should equal vcluster name")
				}).
					WithPolling(constants.PollingInterval).
					WithTimeout(constants.PollingTimeout).
					Should(Succeed(), "Manifest template should be synced")
			})
		})
}
