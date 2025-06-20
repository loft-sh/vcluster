package limitclasses

import (
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Test limitclass on fromHost", ginkgo.Ordered, func() {
	var (
		f                *framework.Framework
		nginxClassName   = "nginx-ingressclass"
		haproxyClassName = "haproxy-ingressclass"

		labelValue1 = "one"
		labelValue2 = "two"

		nginxIngressName   = "web-nginx-ingress"
		haproxyIngressName = "web-haproxy-ingress"

		testNamespace = "default"
		hostNamespace = "vcluster"
	)

	ginkgo.BeforeAll(func() {
		f = framework.DefaultFramework
		ginkgo.By("Creating nginx-ingressclass on host")
		nginxClass := &networkingv1.IngressClass{
			ObjectMeta: metav1.ObjectMeta{
				Name:   nginxClassName,
				Labels: map[string]string{"value": labelValue1},
			},
			Spec: networkingv1.IngressClassSpec{
				Controller: "k8s.io/ingress-nginx",
			},
		}
		_, err := f.HostClient.NetworkingV1().IngressClasses().Create(f.Context, nginxClass, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		ginkgo.By("Creating haproxy-ingressclass on host")
		haproxyClass := &networkingv1.IngressClass{
			ObjectMeta: metav1.ObjectMeta{
				Name:   haproxyClassName,
				Labels: map[string]string{"value": labelValue2},
			},
			Spec: networkingv1.IngressClassSpec{
				Controller: "haproxy.org/ingress-controller",
			},
		}
		_, err = f.HostClient.NetworkingV1().IngressClasses().Create(f.Context, haproxyClass, metav1.CreateOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.AfterAll(func() {
		_ = f.HostClient.NetworkingV1().IngressClasses().Delete(f.Context, nginxClassName, metav1.DeleteOptions{})
		_ = f.HostClient.NetworkingV1().IngressClasses().Delete(f.Context, haproxyClassName, metav1.DeleteOptions{})
		_ = f.HostClient.NetworkingV1().Ingresses(testNamespace).Delete(f.Context, nginxIngressName, metav1.DeleteOptions{})
		_ = f.HostClient.NetworkingV1().Ingresses(testNamespace).Delete(f.Context, haproxyIngressName, metav1.DeleteOptions{})
	})

	ginkgo.It("should only sync ingressClasses with allowed label to vcluster", func() {
		ginkgo.By("Listing all ingresssesClasses available in vcluster")
		gomega.Eventually(func() bool {
			ingressClasses, err := f.VClusterClient.NetworkingV1().IngressClasses().List(f.Context, metav1.ListOptions{})
			if err != nil {
				return false
			}
			for _, ingressClass := range ingressClasses.Items {
				if ingressClass.Name == nginxClassName {
					return true
				}
			}
			return false
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(gomega.BeTrue(), "Timed out waiting for listing all ingressClasses")

		gomega.Consistently(func() bool {
			ingressClasses, err := f.VClusterClient.NetworkingV1().IngressClasses().List(f.Context, metav1.ListOptions{})
			if err != nil {
				return false
			}
			for _, ingressClass := range ingressClasses.Items {
				if ingressClass.Name == haproxyClassName {
					return true
				}
			}
			return false
		}).
			WithPolling(time.Second).
			WithTimeout(framework.PollTimeout).
			Should(gomega.BeFalse(), "Timed out waiting for listing all ingressClasses")
	})

	ginkgo.It("should not sync vcluster ingresses created using an ingressClass not available in vCluster", func() {
		ginkgo.By("Creating a haproxy-ingress using haproxy-ingressclass in vcluster")
		haproxyIngress := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      haproxyIngressName,
				Namespace: testNamespace,
			},
			Spec: networkingv1.IngressSpec{
				IngressClassName: &haproxyClassName,
				Rules: []networkingv1.IngressRule{
					{
						Host: "web-haproxy.local",
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/",
										PathType: func() *networkingv1.PathType { pt := networkingv1.PathTypePrefix; return &pt }(),
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "web",
												Port: networkingv1.ServiceBackendPort{Number: 80},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		_, err := f.VClusterClient.NetworkingV1().Ingresses(testNamespace).Create(f.Context, haproxyIngress, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		ginkgo.By("Ingress should not be synced to host")
		_, err = f.HostClient.NetworkingV1().Ingresses(testNamespace).Get(f.Context, haproxyIngressName, metav1.GetOptions{})
		gomega.Expect(err).To(gomega.HaveOccurred())

		ginkgo.By("There should be a warning message event in the describe of the created ingress")
		gomega.Eventually(func() bool {
			eventList, err := f.VClusterClient.CoreV1().Events(testNamespace).List(f.Context, metav1.ListOptions{})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			for _, event := range eventList.Items {
				if event.InvolvedObject.Kind == "Ingress" && event.InvolvedObject.Name == haproxyIngressName && event.Type == corev1.EventTypeWarning && event.Reason == "SyncWarning" {
					gomega.Expect(event.Message).To(gomega.ContainSubstring(`did not sync ingress "%s" to host because the ingress class "%s" in the host does not match the selector under 'sync.fromHost.ingressClasses.selector'`, haproxyIngressName, haproxyClassName))
					return true
				}
			}
			return false
		}).
			WithTimeout(time.Minute).
			WithPolling(time.Second).
			Should(gomega.BeTrue(), "Timed out waiting for SyncWarning event for ingress %s", haproxyIngressName)
	})

	ginkgo.It("should sync ingresses created in vcluster to host", func() {
		ginkgo.By("Creating a nginx-ingress using nginx-ingressclass in vcluster")
		nginxIngress := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nginxIngressName,
				Namespace: testNamespace,
			},
			Spec: networkingv1.IngressSpec{
				IngressClassName: &nginxClassName,
				Rules: []networkingv1.IngressRule{
					{
						Host: "web-nginx.local",
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/",
										PathType: func() *networkingv1.PathType { pt := networkingv1.PathTypePrefix; return &pt }(),
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "web",
												Port: networkingv1.ServiceBackendPort{Number: 80},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		_, err := f.VClusterClient.NetworkingV1().Ingresses(testNamespace).Create(f.Context, nginxIngress, metav1.CreateOptions{})
		framework.ExpectNoError(err)

		ginkgo.By("should sync ingress created in vCluster to host using ingressClass synced from Host")
		ginkgo.By("Listing all Ingresses in host's vcluster namespace")
		gomega.Eventually(func() bool {
			ingresses, err := f.HostClient.NetworkingV1().Ingresses(hostNamespace).List(f.Context, metav1.ListOptions{})
			if err != nil {
				return false
			}
			for _, ingress := range ingresses.Items {
				if ingress.Name == nginxIngressName+"-x-"+testNamespace+"-x-"+hostNamespace {
					return true
				}
			}
			return false
		}).
			WithTimeout(time.Minute).
			WithPolling(time.Second).
			Should(gomega.BeTrue(), "Timed out waiting for listing all ingresses in host")
	})
})
