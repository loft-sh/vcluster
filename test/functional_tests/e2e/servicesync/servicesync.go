package servicesync

import (
	"context"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

var _ = ginkgo.Describe("map services from host to virtual cluster and vice versa", func() {
	var f *framework.Framework

	ginkgo.JustBeforeEach(func() {
		// use default framework
		f = framework.DefaultFramework
	})

	ginkgo.It("Test service mapping", func() {
		ctx := f.Context

		// make sure physical service doesn't exist initially
		_, err := f.HostClient.CoreV1().Services(f.VclusterNamespace).Get(ctx, "test", metav1.GetOptions{})
		framework.ExpectError(err)
		_, err = f.HostClient.CoreV1().Services("test").Get(ctx, "test", metav1.GetOptions{})
		framework.ExpectError(err)

		// make sure virtual service doesn't exist initially
		_, err = f.VClusterClient.CoreV1().Services("default").Get(ctx, "test", metav1.GetOptions{})
		framework.ExpectError(err)
		_, err = f.VClusterClient.CoreV1().Services("test").Get(ctx, "test", metav1.GetOptions{})
		framework.ExpectError(err)

		// physical -> virtual
		testMapping(ctx, f.HostClient, "test", "test", f.VClusterClient, "default", "test", true)

		// virtual -> physical
		testMapping(ctx, f.VClusterClient, "test", "test", f.HostClient, f.VclusterNamespace, "test", f.MultiNamespaceMode)
	})

	ginkgo.Context("Should sync endpoint updates for a headless service", func() {
		ginkgo.It("in host -> vcluster service mapping", func() {
			checkEndpointsSync(f.Context, f.HostClient, "test", "nginx", f.VClusterClient, "default", "nginx")
		})

		ginkgo.It("in vcluster -> host service mapping", func() {
			checkEndpointsSync(f.Context, f.VClusterClient, "test", "nginx", f.HostClient, f.VclusterNamespace, "nginx")
		})
	})
})

func testMapping(ctx context.Context, fromClient kubernetes.Interface, fromNamespace, fromName string, toClient kubernetes.Interface, toNamespace, toName string, checkEndpoints bool) {
	_, _ = fromClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fromNamespace}}, metav1.CreateOptions{})
	// clean up the namespace
	defer func() {
		err := fromClient.CoreV1().Namespaces().Delete(ctx, fromNamespace, metav1.DeleteOptions{})
		framework.ExpectNoError(err)

		// wait for namespace to be deleted successfully
		waitErr := wait.PollUntilContextTimeout(ctx, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (done bool, err error) {
			_, err = fromClient.CoreV1().Namespaces().Get(ctx, fromNamespace, metav1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return true, nil
				}
				return false, err
			}

			return false, nil
		})
		framework.ExpectNoError(waitErr)
	}()

	// create physical service
	fromService, err := fromClient.CoreV1().Services(fromNamespace).Create(ctx, &corev1.Service{
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
		waitErr := wait.PollUntilContextTimeout(ctx, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (done bool, err error) {
			toService, err = toClient.CoreV1().Services(toNamespace).Get(ctx, toName, metav1.GetOptions{})
			if err != nil {
				if !kerrors.IsNotFound(err) {
					return false, err
				}

				return false, nil
			}

			toEndpoints, err = toClient.CoreV1().Endpoints(toNamespace).Get(ctx, toName, metav1.GetOptions{})
			if err != nil {
				if !kerrors.IsNotFound(err) {
					return false, err
				}

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
		waitErr := wait.PollUntilContextTimeout(ctx, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (done bool, err error) {
			toService, err = toClient.CoreV1().Services(toNamespace).Get(ctx, toName, metav1.GetOptions{})
			if err != nil {
				if !kerrors.IsNotFound(err) {
					return false, err
				}

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
		framework.ExpectEqual(toService.Spec.Selector[translate.MarkerLabel], translate.VClusterName)
		framework.ExpectEqual(toService.Spec.Selector[translate.HostLabel("test")], "test")
	}

	// check service deletion
	err = fromClient.CoreV1().Services(fromNamespace).Delete(ctx, fromService.Name, metav1.DeleteOptions{})
	framework.ExpectNoError(err)

	// verify service gets deleted in vcluster
	waitErr := wait.PollUntilContextTimeout(ctx, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (done bool, err error) {
		_, err = toClient.CoreV1().Services(toNamespace).Get(ctx, toName, metav1.GetOptions{})
		if err == nil {
			return false, nil
		}

		_, err = toClient.CoreV1().Endpoints(toNamespace).Get(ctx, toName, metav1.GetOptions{})
		if err == nil {
			return false, nil
		}

		return true, nil
	})
	framework.ExpectNoError(waitErr)
}

func checkEndpointsSync(ctx context.Context, fromClient kubernetes.Interface, fromNamespace, fromName string, toClient kubernetes.Interface, toNamespace, toName string) {
	var (
		fromEndpoints *corev1.Endpoints
		toEndpoints   *corev1.Endpoints
		deployment    *appsv1.Deployment
		err           error

		two  int32 = 2
		zero int32
	)

	_, err = fromClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fromNamespace}}, metav1.CreateOptions{})
	framework.ExpectNoError(err)

	// clean up the namespace
	defer func() {
		_ = fromClient.CoreV1().Namespaces().Delete(ctx, fromNamespace, metav1.DeleteOptions{})

		// wait for namespace to be deleted successfully
		waitErr := wait.PollUntilContextTimeout(ctx, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (done bool, err error) {
			_, err = fromClient.CoreV1().Namespaces().Get(ctx, fromNamespace, metav1.GetOptions{})
			if err != nil {
				if kerrors.IsNotFound(err) {
					return true, nil
				}
				return false, err
			}

			return false, nil
		})
		framework.ExpectNoError(waitErr)
	}()

	_, err = fromClient.AppsV1().Deployments(fromNamespace).Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: fromName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &two,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nginx",
					Namespace: fromNamespace,
					Labels: map[string]string{
						"app": "nginx",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx",
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{})
	framework.ExpectNoError(err)

	ginkgo.By("Creating the 'from' service targeting the deployment via label selectors")
	_, err = fromClient.CoreV1().Services(fromNamespace).Create(ctx, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: fromName,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "nginx",
			},
			ClusterIP: corev1.ClusterIPNone,
		},
	}, metav1.CreateOptions{})

	ginkgo.By("Ensuring the 'to' service has been created")
	waitErr := wait.PollUntilContextTimeout(ctx, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (done bool, err error) {
		_, err = toClient.CoreV1().Services(toNamespace).Get(ctx, toName, metav1.GetOptions{})
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return false, err
			}
			return false, nil
		}

		return true, nil
	})
	framework.ExpectNoError(err)
	framework.ExpectNoError(waitErr)

	// verify the endpoints
	waitErr = wait.PollUntilContextTimeout(ctx, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (done bool, err error) {
		fromEndpoints, err = fromClient.CoreV1().Endpoints(fromNamespace).Get(ctx, fromName, metav1.GetOptions{})
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return false, err
			}

			return false, nil
		} else if fromEndpoints.Subsets == nil || len(fromEndpoints.Subsets[0].Addresses) != int(two) {
			return false, nil
		}

		return true, nil
	})
	framework.ExpectNoError(err)
	framework.ExpectNoError(waitErr)

	waitErr = wait.PollUntilContextTimeout(ctx, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (done bool, err error) {
		toEndpoints, err = toClient.CoreV1().Endpoints(toNamespace).Get(ctx, toName, metav1.GetOptions{})
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return false, err
			}

			return false, nil
		} else if toEndpoints.Subsets == nil || len(toEndpoints.Subsets[0].Addresses) != int(two) {
			return false, nil
		}

		return true, nil
	})
	framework.ExpectNoError(err)
	framework.ExpectNoError(waitErr)

	ginkgo.By("Ensuring the endpoints are synced")
	framework.ExpectEqual(IPAddresses(fromEndpoints.Subsets[0].Addresses), IPAddresses(toEndpoints.Subsets[0].Addresses))

	ginkgo.By("Scaling down the deployments to 0")
	waitErr = wait.PollUntilContextTimeout(ctx, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (done bool, err error) {
		// scale down the deployment
		deployment, err = fromClient.AppsV1().Deployments(fromNamespace).Get(ctx, fromName, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}

		deployment.Spec.Replicas = &zero

		_, err = fromClient.AppsV1().Deployments(fromNamespace).Update(ctx, deployment, metav1.UpdateOptions{})
		if err != nil {
			if kerrors.IsConflict(err) {
				return false, nil
			}

			return false, err
		}

		return true, nil
	})
	framework.ExpectNoError(err)
	framework.ExpectNoError(waitErr)

	ginkgo.By("Waiting until all endpoints are cleared")
	waitErr = wait.PollUntilContextTimeout(ctx, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (done bool, err error) {
		fromEndpoints, err = fromClient.CoreV1().Endpoints(fromNamespace).Get(ctx, fromName, metav1.GetOptions{})
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return false, err
			}

			return false, nil
		} else if fromEndpoints.Subsets != nil {
			return false, nil
		}

		return true, nil
	})
	framework.ExpectNoError(err)
	framework.ExpectNoError(waitErr)

	ginkgo.By("Scaling up the deployments back")
	waitErr = wait.PollUntilContextTimeout(ctx, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (done bool, err error) {
		// scale up the deployment
		deployment, err = fromClient.AppsV1().Deployments(fromNamespace).Get(ctx, fromName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		deployment.Spec.Replicas = &two

		_, err = fromClient.AppsV1().Deployments(fromNamespace).Update(ctx, deployment, metav1.UpdateOptions{})
		if err != nil {
			return false, err
		}

		return true, nil
	})
	framework.ExpectNoError(err)
	framework.ExpectNoError(waitErr)

	// verify the endpoints
	ginkgo.By("Waiting until all endpoints are registered")
	waitErr = wait.PollUntilContextTimeout(ctx, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (done bool, err error) {
		fromEndpoints, err = fromClient.CoreV1().Endpoints(fromNamespace).Get(ctx, fromName, metav1.GetOptions{})
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return false, err
			}

			return false, nil
		} else if fromEndpoints.Subsets == nil || len(fromEndpoints.Subsets[0].Addresses) != int(two) {
			return false, nil
		}

		return true, nil
	})
	framework.ExpectNoError(err)
	framework.ExpectNoError(waitErr)

	waitErr = wait.PollUntilContextTimeout(ctx, time.Millisecond*500, framework.PollTimeout, true, func(ctx context.Context) (done bool, err error) {
		toEndpoints, err = toClient.CoreV1().Endpoints(toNamespace).Get(ctx, toName, metav1.GetOptions{})
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return false, err
			}

			return false, nil
		} else if toEndpoints.Subsets == nil || len(toEndpoints.Subsets[0].Addresses) != int(two) {
			return false, nil
		}

		return true, nil
	})
	framework.ExpectNoError(err)
	framework.ExpectNoError(waitErr)

	ginkgo.By("Ensuring the endpoints are still synced")
	framework.ExpectEqual(IPAddresses(fromEndpoints.Subsets[0].Addresses), IPAddresses(toEndpoints.Subsets[0].Addresses))
}

func IPAddresses(endpointAddresses []corev1.EndpointAddress) []string {
	xIP := make([]string, 0)

	for _, address := range endpointAddresses {
		xIP = append(xIP, address.IP)
	}

	return xIP
}
