package services

import (
	"encoding/json"
	"fmt"

	"github.com/loft-sh/vcluster/e2e/framework"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Services are created as expected", func() {
	var (
		f         *framework.Framework
		iteration int
		ns        string
	)

	ginkgo.JustBeforeEach(func() {
		// use default framework
		f = framework.DefaultFramework
		iteration++
		ns = fmt.Sprintf("e2e-syncer-services-%d", iteration)
		// execute cleanup in case previous e2e test were terminated prematurely
		err := f.DeleteTestNamespace(ns, true)
		framework.ExpectNoError(err)

		// create test namespace
		_, err = f.VclusterClient.CoreV1().Namespaces().Create(f.Context, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}, metav1.CreateOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.AfterEach(func() {
		// delete test namespace
		err := f.DeleteTestNamespace(ns, false)
		framework.ExpectNoError(err)
	})

	ginkgo.It("Test Service gets created when no Kind is present in body", func() {
		service := corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "myservice",
				Namespace: ns,
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{"doesnt": "matter"},
				Ports: []corev1.ServicePort{
					{Port: 80},
				},
			},
		}
		body, err := json.Marshal(service)
		framework.ExpectNoError(err)

		_, err = f.VclusterClient.RESTClient().Post().AbsPath("/api/v1/namespaces/" + ns + "/services").Body(body).DoRaw(f.Context)
		framework.ExpectNoError(err)

		_, err = f.VclusterClient.CoreV1().Services(ns).Get(f.Context, service.Name, metav1.GetOptions{})
		framework.ExpectNoError(err)

		_, err = f.HostClient.CoreV1().Services(f.VclusterNamespace).Get(f.Context, translate.ObjectPhysicalName(&service), metav1.GetOptions{})
		framework.ExpectNoError(err)
	})
})
