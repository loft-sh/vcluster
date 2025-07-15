package fromhost

import (
	"time"

	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/reference"
)

var _ = ginkgo.Describe("Events can be force synced using an annotation", ginkgo.Ordered, func() {
	var (
		f              *framework.Framework
		event1         *corev1.Event
		dummyConfigMap *corev1.ConfigMap
		event1Name     = "dummy"
		event1Message  = "test msg"
	)

	ginkgo.BeforeAll(func() {
		f = framework.DefaultFramework
		event1 = &corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      event1Name,
				Namespace: f.VClusterNamespace,
				Annotations: map[string]string{
					"vcluster.loft.sh/force-sync": "true",
				},
			},
			Message: event1Message,
		}
		dummyConfigMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dummy-unrelated",
				Namespace: f.VClusterNamespace,
			},
		}
	})

	ginkgo.AfterAll(func() {
		framework.ExpectNoError(f.HostClient.CoreV1().Events(event1.GetNamespace()).Delete(f.Context, event1.GetName(), metav1.DeleteOptions{}))
		framework.ExpectNoError(f.HostClient.CoreV1().ConfigMaps(dummyConfigMap.GetNamespace()).Delete(f.Context, dummyConfigMap.GetName(), metav1.DeleteOptions{}))
	})

	ginkgo.It("Secrets are synced to virtual cluster", func() {
		involvedObj, err := f.HostClient.CoreV1().ConfigMaps(event1.GetNamespace()).Create(f.Context, dummyConfigMap, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		ref, err := reference.GetReference(scheme.Scheme, involvedObj)
		framework.ExpectNoError(err)
		event1.InvolvedObject = *ref

		_, err = f.HostClient.CoreV1().Events(event1.GetNamespace()).Create(f.Context, event1, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		gomega.Eventually(func(g gomega.Gomega) {
			virtual1, err := f.VClusterClient.CoreV1().Events("default").Get(f.Context, event1Name, metav1.GetOptions{})
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(virtual1.Message).To(gomega.Equal(event1.Message))
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout / 4).
			Should(gomega.Succeed())
	})

})
