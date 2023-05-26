package manifests

import (
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/controllers/manifests"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	ChartName         = "ingress-nginx"
	ChartNamespace    = "ingress-nginx"
	ChartOCIName      = "prometheus"
	ChartOCINamespace = "prometheus"
)

var _ = ginkgo.Describe("Helm charts (regular and OCI) are synced and applied as expected", func() {
	var (
		f                 *framework.Framework
		hostConfigMapName string
		HelmSecretLabels  = map[string]string{
			"owner": "helm",
			"name":  ChartName,
		}
		HelmOCIDeploymentLabels = map[string]string{
			"app":     "prometheus",
			"release": ChartOCIName,
		}
	)

	ginkgo.JustBeforeEach(func() {
		f = framework.DefaultFramework
		hostConfigMapName = fmt.Sprintf("%s-%s", f.VclusterNamespace, InitConfigmapSuffix)
	})

	ginkgo.It("Test if configmap for both charts is created as expected", func() {
		_, err := f.HostClient.
			CoreV1().
			ConfigMaps(f.VclusterNamespace).
			Get(f.Context, hostConfigMapName, metav1.GetOptions{})
		framework.ExpectNoError(err)
	})


	ginkgo.It("Test if configmap for both charts gets applied", func() {
    // ignore deprecation notice due to https://github.com/kubernetes/kubernetes/issues/116712
		//nolint:staticcheck
		err := wait.PollImmediate(time.Millisecond*500, framework.PollTimeout, func() (bool, error) {
			cm, err := f.HostClient.CoreV1().ConfigMaps(f.VclusterNamespace).
				Get(f.Context, hostConfigMapName, metav1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}

			status := manifests.ParseStatus(cm)
			// validate that all charts are Success
			for _, chart := range status.Charts {
				if chart.Phase != string(manifests.StatusSuccess) {
					return false, nil
				}
			}

			return status.Phase == string(manifests.StatusSuccess) && len(status.Charts) == 2, nil
		})

		framework.ExpectNoError(err)
	})

	ginkgo.It("Test nginx release secret existence in vcluster (regular chart)", func() {
    // ignore deprecation notice due to https://github.com/kubernetes/kubernetes/issues/116712
		//nolint:staticcheck
		err := wait.PollImmediate(time.Millisecond*500, framework.PollTimeout, func() (bool, error) {
			secList, err := f.VclusterClient.CoreV1().Secrets(ChartNamespace).List(f.Context, metav1.ListOptions{
				LabelSelector: labels.SelectorFromSet(HelmSecretLabels).String(),
			})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}

			for _, sec := range secList.Items {
				_, ok := sec.Data["release"]
				if ok {
					return true, nil
				}
			}

			return false, nil
		})

		framework.ExpectNoError(err)
	})

	ginkgo.It("Test prometheus release deployment existence in vcluster (OCI chart)", func() {
		err := wait.PollImmediate(time.Millisecond*500, framework.PollTimeout, func() (bool, error) {
			deployList, err := f.VclusterClient.AppsV1().Deployments(ChartOCINamespace).List(f.Context, metav1.ListOptions{
				LabelSelector: labels.SelectorFromSet(HelmOCIDeploymentLabels).String(),
			})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return false, nil
				}
				return false, err
			}
			// return OK if a deployment is found
			if len(deployList.Items) == 1 {
				return true, nil
			}

			return false, nil
		})

		framework.ExpectNoError(err)
	})
})
