package manifests

import (
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TestManifestName      = "test-configmap"
	TestManifestName2     = "test-configmap-2"
	TestManifestNamespace = "default"
)

var _ = ginkgo.Describe("Init manifests are synced and applied as expected", func() {
	f := framework.DefaultFramework

	ginkgo.It("Test if manifest is synced with the vcluster", func() {
		err := f.WaitForInitManifestConfigMapCreation(TestManifestName, TestManifestNamespace)
		framework.ExpectNoError(err)

		manifest, err := f.VClusterClient.CoreV1().ConfigMaps(TestManifestNamespace).Get(f.Context, TestManifestName, metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectHaveKey(manifest.Data, "foo", "modified init manifest is supposed to have the foo key")
		framework.ExpectEqual(manifest.Data["foo"], "bar")
	})

	ginkgo.It("Test if manifest template is synced with the vcluster", func() {
		err := f.WaitForInitManifestConfigMapCreation(TestManifestName2, TestManifestNamespace)
		framework.ExpectNoError(err)

		manifest, err := f.VClusterClient.CoreV1().ConfigMaps(TestManifestNamespace).Get(f.Context, TestManifestName2, metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectHaveKey(manifest.Data, "foo", "modified init manifest is supposed to have the foo key")
		framework.ExpectEqual(manifest.Data["foo"], "vcluster")
	})
})
