package servicesync

import (
	"context"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

var _ = ginkgo.Describe("map services from host to virtual cluster and vice versa", func() {
	var (
		f *framework.Framework
	)

	ginkgo.JustBeforeEach(func() {
		// use default framework
		f = framework.DefaultFramework
	})

	ginkgo.It("Test service mapping", func() {
		// make sure physical service doesn't exist initially
		_, err := f.HostClient.CoreV1().Services(f.VclusterNamespace).Get(context.Background(), "test", metav1.GetOptions{})
		framework.ExpectError(err)
		_, err = f.HostClient.CoreV1().Services("test").Get(context.Background(), "test", metav1.GetOptions{})
		framework.ExpectError(err)

		// make sure virtual service doesn't exist initially
		_, err = f.VclusterClient.CoreV1().Services("default").Get(context.Background(), "test", metav1.GetOptions{})
		framework.ExpectError(err)
		_, err = f.VclusterClient.CoreV1().Services("test").Get(context.Background(), "test", metav1.GetOptions{})
		framework.ExpectError(err)

		// physical -> virtual
		testMapping(f.HostClient, "test", "test", f.VclusterClient, "default", "test", true)

		// virtual -> physical
		testMapping(f.VclusterClient, "test", "test", f.HostClient, f.VclusterNamespace, "test", false)
	})
})

func testMapping(fromClient kubernetes.Interface, fromNamespace, fromName string, toClient kubernetes.Interface, toNamespace, toName string, checkEndpoints bool) {
	// create physical service
	_, _ = fromClient.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fromName}}, metav1.CreateOptions{})
	fromService, err := fromClient.CoreV1().Services(fromNamespace).Create(context.Background(), &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fromName,
			Namespace: fromNamespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"test": "test",
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "custom",
					Port:     8080,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}, metav1.CreateOptions{})
	framework.ExpectNoError(err)

	// check endpoints?
	if checkEndpoints {
		// wait for vcluster service
		var (
			toEndpoints *corev1.Endpoints
			toService   *corev1.Service
		)
		waitErr := wait.PollImmediate(time.Millisecond*100, time.Second*10, func() (done bool, err error) {
			toService, err = toClient.CoreV1().Services(toNamespace).Get(context.Background(), toName, metav1.GetOptions{})
			if err != nil {
				return false, nil
			}

			toEndpoints, err = toClient.CoreV1().Endpoints(toNamespace).Get(context.Background(), toName, metav1.GetOptions{})
			if err != nil {
				return false, nil
			}

			return true, nil
		})
		framework.ExpectNoError(err)
		framework.ExpectNoError(waitErr)

		// check if endpoints are correct
		framework.ExpectEqual(len(toEndpoints.Subsets), 1)
		framework.ExpectEqual(len(toEndpoints.Subsets[0].Addresses), 1)
		framework.ExpectEqual(toEndpoints.Subsets[0].Addresses[0].IP, fromService.Spec.ClusterIP)
		framework.ExpectEqual(len(toService.Spec.Ports), 1)
		framework.ExpectEqual(toService.Spec.Ports[0].Name, "custom")
		framework.ExpectEqual(toService.Spec.Ports[0].Port, int32(8080))
	} else {
		// wait for vcluster service
		var toService *corev1.Service
		waitErr := wait.PollImmediate(time.Millisecond*100, time.Second*10, func() (done bool, err error) {
			toService, err = toClient.CoreV1().Services(toNamespace).Get(context.Background(), toName, metav1.GetOptions{})
			if err != nil {
				return false, nil
			}

			return true, nil
		})
		framework.ExpectNoError(err)
		framework.ExpectNoError(waitErr)

		// check if service is correct
		framework.ExpectEqual(len(toService.Spec.Ports), 1)
		framework.ExpectEqual(toService.Spec.Ports[0].Name, "custom")
		framework.ExpectEqual(toService.Spec.Ports[0].Port, int32(8080))
		framework.ExpectEqual(len(toService.Spec.Selector), 3)
		framework.ExpectEqual(toService.Spec.Selector[translate.NamespaceLabel], fromNamespace)
		framework.ExpectEqual(toService.Spec.Selector[translate.MarkerLabel], translate.Suffix)
		framework.ExpectEqual(toService.Spec.Selector[translate.ConvertLabelKeyWithPrefix(translate.LabelPrefix, "test")], "test")
	}

	// check service deletion
	err = fromClient.CoreV1().Services(fromNamespace).Delete(context.Background(), fromService.Name, metav1.DeleteOptions{})
	framework.ExpectNoError(err)

	// verify service gets deleted in vcluster
	waitErr := wait.PollImmediate(time.Millisecond*100, time.Second*10, func() (done bool, err error) {
		_, err = toClient.CoreV1().Services(toNamespace).Get(context.Background(), toName, metav1.GetOptions{})
		if err == nil {
			return false, nil
		}

		_, err = toClient.CoreV1().Endpoints(toNamespace).Get(context.Background(), toName, metav1.GetOptions{})
		if err == nil {
			return false, nil
		}

		return true, nil
	})
	framework.ExpectNoError(waitErr)
}
