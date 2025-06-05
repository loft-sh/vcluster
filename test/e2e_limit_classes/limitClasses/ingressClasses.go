package limitclasses

import (
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
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
		// Create nginx-ingressclass on host
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

		// Create haproxy-ingressclass on host
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
		gomega.Eventually(func() []string {
			ics, err := f.VClusterClient.NetworkingV1().IngressClasses().List(f.Context, metav1.ListOptions{}) // List all ingressClasses in the vCluster
			if err != nil {
				return nil
			}
			var names []string
			for _, ic := range ics.Items {
				names = append(names, ic.Name)
			}
			return names
		}).WithTimeout(time.Minute).WithPolling(time.Second).
			Should(gomega.ContainElement(nginxClassName))

		gomega.Consistently(func() []string {
			ics, err := f.VClusterClient.NetworkingV1().IngressClasses().List(f.Context, metav1.ListOptions{})
			if err != nil {
				return nil
			}
			var names []string
			for _, ic := range ics.Items {
				names = append(names, ic.Name)
			}
			return names
		}).WithTimeout(5 * time.Second).WithPolling(time.Second).
			ShouldNot(gomega.ContainElement(haproxyClassName))
	})

	ginkgo.It("should not sync vcluster ingresses using a filtered ingressClass to host", func() {
		// Try to create a haproxy-ingress using haproxy-ingressclass in vcluster
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

		// It should NOT be synced to host
		time.Sleep(5 * time.Second)
		_, err = f.HostClient.NetworkingV1().Ingresses(testNamespace).Get(f.Context, haproxyIngressName, metav1.GetOptions{})
		gomega.Expect(err).To(gomega.HaveOccurred())
	})

	ginkgo.It("should sync vcluster ingresses using allowed ingressClass to host", func() {
		// Create nginx-ingress using nginx-ingressclass in vcluster
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

		// It should be synced to host
		gomega.Eventually(func() []string {
			igs, err := f.HostClient.NetworkingV1().Ingresses(hostNamespace).List(f.Context, metav1.ListOptions{}) // List all ingresses in the vCluster
			if err != nil {
				return nil
			}
			var names []string
			for _, igc := range igs.Items {
				names = append(names, igc.Name)
			}
			return names
		}).WithTimeout(time.Minute).WithPolling(time.Second).
			Should(gomega.ContainElement(nginxIngressName + "-x-" + testNamespace + "-x-" + hostNamespace))
	})
})
