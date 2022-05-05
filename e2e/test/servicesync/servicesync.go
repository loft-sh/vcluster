package servicesync

import (
	"context"
	"github.com/loft-sh/vcluster/e2e/framework"
	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"time"
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
		testMapping(f.HostClient, "test", "test", f.VclusterClient, "default", "test")

		// virtual -> physical
		testMapping(f.VclusterClient, "test", "test", f.HostClient, f.VclusterNamespace, "test")
	})
})

func testMapping(fromClient kubernetes.Interface, fromNamespace, fromName string, toClient kubernetes.Interface, toNamespace, toName string) {
	// create physical service
	_, _ = fromClient.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fromName}}, metav1.CreateOptions{})
	fromService, err := fromClient.CoreV1().Services(fromNamespace).Create(context.Background(), &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fromName,
			Namespace: fromNamespace,
		},
		Spec: corev1.ServiceSpec{
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

	// wait for vcluster service
	var toEndpoints *corev1.Endpoints
	waitErr := wait.PollImmediate(time.Millisecond*100, time.Second*10, func() (done bool, err error) {
		_, err = toClient.CoreV1().Services(toNamespace).Get(context.Background(), toName, metav1.GetOptions{})
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

	// check service deletion
	err = fromClient.CoreV1().Services(fromNamespace).Delete(context.Background(), fromService.Name, metav1.DeleteOptions{})
	framework.ExpectNoError(err)

	// verify service gets deleted in vcluster
	waitErr = wait.PollImmediate(time.Millisecond*100, time.Second*10, func() (done bool, err error) {
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
