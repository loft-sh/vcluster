package manifests

import (
	"fmt"

	"github.com/loft-sh/vcluster/e2e/framework"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TestManifestName      = "test-configmap"
	TestManifestNamespace = "default"

	InitManifestConfigmapSuffix = "init-manifests"
)

var _ = ginkgo.Describe("Init manifests are synced and applied as expected", func() {
	var (
		f         *framework.Framework
		iteration int
		ns        string
	)

	ginkgo.JustBeforeEach(func() {
		f = framework.DefaultFramework
		iteration++
		ns = fmt.Sprintf("e2e-init-manifests-%d-%s", iteration, random.RandomString(5))

		_, err := f.VclusterClient.CoreV1().Namespaces().Create(f.Context, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			}}, metav1.CreateOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.AfterEach(func() {
		err := f.DeleteTestNamespace(ns, false)
		framework.ExpectNoError(err)

		// reset configmap to be empty
		initmanifests, err := f.HostClient.
			CoreV1().
			ConfigMaps(f.VclusterNamespace).
			Get(f.Context, fmt.Sprintf("%s-%s", f.VclusterNamespace, InitManifestConfigmapSuffix), metav1.GetOptions{})
		framework.ExpectNoError(err)

		initmanifests.Data["initmanifests.yaml"] = "---"
		_, err = f.HostClient.CoreV1().ConfigMaps(f.VclusterNamespace).Update(f.Context, initmanifests, metav1.UpdateOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.It("Test if init manifests are initially empty", func() {
		initmanifests, err := f.HostClient.
			CoreV1().
			ConfigMaps(f.VclusterNamespace).
			Get(f.Context, fmt.Sprintf("%s-%s", f.VclusterNamespace, InitManifestConfigmapSuffix), metav1.GetOptions{})
		framework.ExpectNoError(err)

		framework.ExpectEqual(initmanifests.Data["initmanifests.yaml"], "---")
	})

	ginkgo.It("Test if manifest is synced with the vcluster", func() {
		initmanifests, err := f.HostClient.
			CoreV1().
			ConfigMaps(f.VclusterNamespace).
			Get(f.Context, fmt.Sprintf("%s-%s", f.VclusterNamespace, InitManifestConfigmapSuffix), metav1.GetOptions{})
		framework.ExpectNoError(err)

		testManifest := fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s
data:
  foo: bar
`, TestManifestName)

		initmanifests.Data["initmanifests.yaml"] = testManifest
		_, err = f.HostClient.CoreV1().ConfigMaps(f.VclusterNamespace).Update(f.Context, initmanifests, metav1.UpdateOptions{})
		framework.ExpectNoError(err)

		err = f.WaitForInitManifestConfigMapCreation(TestManifestName, TestManifestNamespace)
		framework.ExpectNoError(err)

		manifest, err := f.VclusterClient.CoreV1().ConfigMaps(TestManifestNamespace).Get(f.Context, TestManifestName, metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectHaveKey(manifest.Data, "foo", "modified init manifest is supposed to have the foo key")
		framework.ExpectEqual(manifest.Data["foo"], "bar")
	})
})
