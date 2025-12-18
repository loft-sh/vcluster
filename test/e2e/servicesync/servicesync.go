package servicesync

import (
	"context"
	"fmt"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

var _ = ginkgo.Describe("Verify mapping and syncing of services and endpoints", ginkgo.Ordered, func() {
	var (
		f           *framework.Framework
		testService *corev1.Service
		//nolint:staticcheck // SA1019: corev1.Endpoints is deprecated, but still required for compatibility
		testEndpoint     *corev1.Endpoints
		serviceName      = "test-service-sync"
		serviceNamespace = "default"
		endpointName     = "test-service-sync"
	)
	ginkgo.BeforeAll(func() {
		f = framework.DefaultFramework
		testService = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: serviceNamespace,
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: "None",
				Ports: []corev1.ServicePort{
					{
						Name:       "custom-port",
						Port:       8080,
						Protocol:   corev1.ProtocolTCP,
						TargetPort: intstr.FromInt(5000),
					},
				},
			},
		}
		//nolint:staticcheck // SA1019: corev1.Endpoints is deprecated, but still required for compatibility
		testEndpoint = &corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      endpointName,
				Namespace: serviceNamespace,
			},
			//nolint:staticcheck // SA1019: corev1.Endpoints is deprecated, but still required for compatibility
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{
							IP: "1.1.1.1",
						},
					},
					Ports: []corev1.EndpointPort{
						{
							Port: 5000,
						},
					},
				},
			},
		}
	})

	ginkgo.AfterAll(func() {
		err := f.VClusterClient.CoreV1().Endpoints(serviceNamespace).Delete(f.Context, endpointName, metav1.DeleteOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		}
		err = f.VClusterClient.CoreV1().Services(serviceNamespace).Delete(f.Context, serviceName, metav1.DeleteOptions{})
		if err != nil && !kerrors.IsNotFound(err) {
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		}
	})

	ginkgo.It("Test service mapping", func() {
		ctx := f.Context

		// make sure physical service doesn't exist initially
		_, err := f.HostClient.CoreV1().Services(f.VClusterNamespace).Get(ctx, "test", metav1.GetOptions{})
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
		testMapping(ctx, f.VClusterClient, "test", "test", f.HostClient, f.VClusterNamespace, "test", false)
	})

	ginkgo.Context("Should sync endpoint updates for a headless service", func() {
		ginkgo.It("in host -> vcluster service mapping", func() {
			checkEndpointsSync(f.Context, f.HostClient, "test", "nginx", f.VClusterClient, "default", "nginx")
		})

		ginkgo.It("in vcluster -> host service mapping", func() {
			checkEndpointsSync(f.Context, f.VClusterClient, "test", "nginx", f.HostClient, f.VClusterNamespace, "nginx")
		})
	})

	ginkgo.Context("Verify endpoint sync when endpoint is deployed before service", func() {
		ginkgo.It("Should sync Service, Endpoints, and EndpointSlice from vCluster to host cluster", func() {
			ginkgo.By("Create Service Endpoint in vCluster")
			_, err := f.VClusterClient.CoreV1().Endpoints(serviceNamespace).Create(f.Context, testEndpoint, metav1.CreateOptions{})
			framework.ExpectNoError(err)

			ginkgo.By("Create Service in vCluster")
			_, err = f.VClusterClient.CoreV1().Services(serviceNamespace).Create(f.Context, testService, metav1.CreateOptions{})
			framework.ExpectNoError(err)

			ginkgo.By("Verify Endpoint exists in vCluster")
			_, err = f.VClusterClient.CoreV1().Endpoints(serviceNamespace).Get(f.Context, endpointName, metav1.GetOptions{})
			framework.ExpectNoError(err)

			ginkgo.By("Verify Service exists in vCluster")
			_, err = f.VClusterClient.CoreV1().Services(serviceNamespace).Get(f.Context, serviceName, metav1.GetOptions{})
			framework.ExpectNoError(err)

			ginkgo.By("Verify EndpointSlice exists in vCluster")
			gomega.Eventually(func(g gomega.Gomega) {
				vclusterEndpointSlice, err := f.VClusterClient.DiscoveryV1().EndpointSlices(serviceNamespace).List(f.Context, metav1.ListOptions{
					LabelSelector: fmt.Sprintf("kubernetes.io/service-name=%s", serviceName),
				})
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(vclusterEndpointSlice.Items).To(gomega.HaveLen(1))
			}).WithPolling(time.Second).WithTimeout(framework.PollTimeout).Should(gomega.Succeed())

			translatedServiceName := translate.SingleNamespaceHostName(serviceName, serviceNamespace, translate.VClusterName)

			ginkgo.By("Verify Service exists in Host Cluster")
			gomega.Eventually(func(g gomega.Gomega) {
				hostService, err := f.HostClient.CoreV1().Services(f.VClusterNamespace).Get(f.Context, translatedServiceName, metav1.GetOptions{})
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(hostService.Spec.Ports).To(gomega.HaveLen(1))
				if len(hostService.Spec.Ports) > 0 {
					g.Expect(hostService.Spec.Ports[0].Name).To(gomega.Equal("custom-port"))
					g.Expect(hostService.Spec.Ports[0].Port).To(gomega.Equal(int32(8080)))
				}
			}).WithPolling(time.Second).WithTimeout(framework.PollTimeout).Should(gomega.Succeed())

			ginkgo.By("Verify Endpoint exists in Host Cluster")
			gomega.Eventually(func(g gomega.Gomega) {
				hostEndpoint, err := f.HostClient.CoreV1().Endpoints(f.VClusterNamespace).Get(f.Context, translatedServiceName, metav1.GetOptions{})
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(hostEndpoint.Subsets).To(gomega.HaveLen(1))
				if len(hostEndpoint.Subsets) > 0 {
					g.Expect(hostEndpoint.Subsets[0].Addresses).To(gomega.HaveLen(1))
					if len(hostEndpoint.Subsets[0].Addresses) > 0 {
						g.Expect(hostEndpoint.Subsets[0].Addresses[0].IP).To(gomega.Equal("1.1.1.1"))
					}
					g.Expect(hostEndpoint.Subsets[0].Ports).To(gomega.HaveLen(1))
					if len(hostEndpoint.Subsets[0].Ports) > 0 {
						g.Expect(hostEndpoint.Subsets[0].Ports[0].Port).To(gomega.Equal(int32(5000)))
					}
				}
			}).WithPolling(time.Second).WithTimeout(framework.PollTimeout).Should(gomega.Succeed())

			ginkgo.By("Verify EndpointSlice exists in Host Cluster")
			gomega.Eventually(func(g gomega.Gomega) {
				hostEndpointSlice, err := f.HostClient.DiscoveryV1().EndpointSlices(f.VClusterNamespace).List(f.Context, metav1.ListOptions{
					LabelSelector: fmt.Sprintf("kubernetes.io/service-name=%s", translatedServiceName),
				})
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(hostEndpointSlice.Items).To(gomega.HaveLen(1))
			}).WithPolling(time.Second).WithTimeout(framework.PollTimeout).Should(gomega.Succeed())

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
			//nolint:staticcheck // SA1019: corev1.Endpoints is deprecated, but still required for compatibility
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
		framework.ExpectEqual(toService.Spec.ClusterIP, fromService.Spec.ClusterIP)
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
		//nolint:staticcheck // SA1019: corev1.Endpoints is deprecated, but still required for compatibility
		fromEndpoints *corev1.Endpoints
		//nolint:staticcheck // SA1019: corev1.Endpoints is deprecated, but still required for compatibility
		toEndpoints *corev1.Endpoints
		deployment  *appsv1.Deployment
		err         error

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
