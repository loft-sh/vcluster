package fromhost

import (
	"reflect"
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Events can be force synced using an annotation", ginkgo.Ordered, func() {
	var (
		f             *framework.Framework
		event1        *corev1.Event
		event1Name    = "dummy"
		event1Message = "test msg"
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
	})

	ginkgo.AfterAll(func() {
		framework.ExpectNoError(f.HostClient.CoreV1().Events(event1.GetNamespace()).Delete(f.Context, event1.GetName(), metav1.DeleteOptions{}))
	})

	ginkgo.It("Secrets are synced to virtual cluster", func() {
		_, err := f.HostClient.CoreV1().Events(event1.GetNamespace()).Create(f.Context, event1, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		gomega.Eventually(func() bool {
			virtual1, err := f.VClusterClient.CoreV1().Events("default").Get(f.Context, event1Name, metav1.GetOptions{})
			if err != nil {
				return false
			}
			if !reflect.DeepEqual(virtual1.Message, event1.Message) {
				f.Log.Errorf("expected %#v in virtual.Message got %#v", event1.Message, virtual1.Message)
				return false
			}
			return true
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout / 4).
			Should(gomega.BeTrue())
	})

})
