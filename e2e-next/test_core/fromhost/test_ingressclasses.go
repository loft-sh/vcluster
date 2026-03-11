package fromhost

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var _ = Describe("IngressClasses sync from host",
	Ordered,
	labels.Core,
	labels.Sync,
	cluster.Use(clusters.IngressClassesVCluster),
	cluster.Use(clusters.HostCluster),
	func() {
		var (
			hostClient    kubernetes.Interface
			vClusterClient kubernetes.Interface
			vClusterName  = clusters.IngressClassesVClusterName

			// nginxClassName and haproxyClassName are fixed names used to create
			// IngressClasses on the host. The vcluster selector is configured to
			// match only the label "value: one" (nginx), so haproxy should not sync.
			nginxClassName   = "nginx-ingressclass"
			haproxyClassName = "haproxy-ingressclass"

			labelValue1 = "one"
			labelValue2 = "two"

			nginxIngressName   = "web-nginx-ingress"
			haproxyIngressName = "web-haproxy-ingress"

			testNamespace string
			hostNamespace string
		)

		BeforeAll(func(ctx context.Context) context.Context {
			hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
			Expect(hostClient).NotTo(BeNil())
			vClusterClient = cluster.CurrentKubeClientFrom(ctx)
			Expect(vClusterClient).NotTo(BeNil())

			testNamespace = "default"
			hostNamespace = "vcluster-" + vClusterName

			By("creating nginx-ingressclass on host")
			nginxClass := &networkingv1.IngressClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:   nginxClassName,
					Labels: map[string]string{"value": labelValue1},
				},
				Spec: networkingv1.IngressClassSpec{
					Controller: "k8s.io/ingress-nginx",
				},
			}
			_, err := hostClient.NetworkingV1().IngressClasses().Create(ctx, nginxClass, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func(ctx context.Context) {
				err := hostClient.NetworkingV1().IngressClasses().Delete(ctx, nginxClassName, metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			By("creating haproxy-ingressclass on host")
			haproxyClass := &networkingv1.IngressClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:   haproxyClassName,
					Labels: map[string]string{"value": labelValue2},
				},
				Spec: networkingv1.IngressClassSpec{
					Controller: "haproxy.org/ingress-controller",
				},
			}
			_, err = hostClient.NetworkingV1().IngressClasses().Create(ctx, haproxyClass, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func(ctx context.Context) {
				err := hostClient.NetworkingV1().IngressClasses().Delete(ctx, haproxyClassName, metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			return ctx
		})

		It("should only sync ingressClasses with allowed label to vcluster", func(ctx context.Context) {
			By("listing all ingressClasses available in vcluster")
			Eventually(func(g Gomega) {
				ingressClasses, err := vClusterClient.NetworkingV1().IngressClasses().List(ctx, metav1.ListOptions{})
				g.Expect(err).NotTo(HaveOccurred())

				foundNginxIngressClass := false
				foundHaproxyIngressClass := false
				for _, ingressClass := range ingressClasses.Items {
					if ingressClass.Name == nginxClassName {
						foundNginxIngressClass = true
					}
					if ingressClass.Name == haproxyClassName {
						foundHaproxyIngressClass = true
					}
				}
				g.Expect(foundNginxIngressClass).To(BeTrue(), "nginx ingress class should be synced to vcluster")
				g.Expect(foundHaproxyIngressClass).To(BeFalse(), "haproxy ingress class should not be synced to vcluster")
			}).
				WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeout).
				Should(Succeed(), "timed out waiting for ingressClasses in vcluster")
		})

		It("should not sync vcluster ingresses created using an ingressClass not available in vcluster", func(ctx context.Context) {
			By("creating a haproxy-ingress using haproxy-ingressclass in vcluster")
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
			_, err := vClusterClient.NetworkingV1().Ingresses(testNamespace).Create(ctx, haproxyIngress, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func(ctx context.Context) {
				err := vClusterClient.NetworkingV1().Ingresses(testNamespace).Delete(ctx, haproxyIngressName, metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			By("verifying ingress is not synced to host")
			_, err = hostClient.NetworkingV1().Ingresses(testNamespace).Get(ctx, haproxyIngressName, metav1.GetOptions{})
			Expect(err).To(HaveOccurred())

			By("waiting for a SyncWarning event on the ingress in vcluster")
			Eventually(func(g Gomega) {
				eventList, err := vClusterClient.CoreV1().Events(testNamespace).List(ctx, metav1.ListOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				found := false
				for _, event := range eventList.Items {
					if event.InvolvedObject.Kind == "Ingress" &&
						event.InvolvedObject.Name == haproxyIngressName &&
						event.Type == corev1.EventTypeWarning &&
						event.Reason == "SyncWarning" {
						g.Expect(event.Message).To(ContainSubstring(
							`did not sync ingress "%s" to host because the ingress class "%s" in the host does not match the selector under 'sync.fromHost.ingressClasses.selector'`,
							haproxyIngressName, haproxyClassName,
						))
						found = true
					}
				}
				g.Expect(found).To(BeTrue(), "expected SyncWarning event for ingress %s", haproxyIngressName)
			}).
				WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeout).
				Should(Succeed(), "timed out waiting for SyncWarning event for ingress %s", haproxyIngressName)
		})

		It("should sync ingresses created in vcluster using an allowed ingressClass to host", func(ctx context.Context) {
			By("creating a nginx-ingress using nginx-ingressclass in vcluster")
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
			_, err := vClusterClient.NetworkingV1().Ingresses(testNamespace).Create(ctx, nginxIngress, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func(ctx context.Context) {
				err := vClusterClient.NetworkingV1().Ingresses(testNamespace).Delete(ctx, nginxIngressName, metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			By("waiting for nginx ingress to be synced to host vcluster namespace")
			// The synced ingress name on the host follows the pattern:
			// <name>-x-<namespace>-x-<host-namespace>
			expectedHostIngressName := nginxIngressName + "-x-" + testNamespace + "-x-" + hostNamespace
			Eventually(func(g Gomega) {
				ingresses, err := hostClient.NetworkingV1().Ingresses(hostNamespace).List(ctx, metav1.ListOptions{})
				g.Expect(err).NotTo(HaveOccurred())
				found := false
				for _, ingress := range ingresses.Items {
					if ingress.Name == expectedHostIngressName {
						found = true
					}
				}
				g.Expect(found).To(BeTrue(), "expected ingress %s in host namespace %s", expectedHostIngressName, hostNamespace)
			}).
				WithPolling(constants.PollingInterval).
				WithTimeout(constants.PollingTimeout).
				Should(Succeed(), "timed out waiting for nginx ingress to appear on host")
		})
	},
)
