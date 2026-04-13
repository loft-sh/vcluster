package fromhost

import (
	"context"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// FromHostIngressClassesSpec registers ingressClass sync from host tests.
func FromHostIngressClassesSpec() {
	Describe("IngressClasses sync from host",
		labels.PR,
		labels.Core,
		labels.Sync,
		labels.IngressClasses,
		func() {
			var (
				hostClient     kubernetes.Interface
				vClusterClient kubernetes.Interface
				vClusterName   string
				vClusterHostNS string
			)

			BeforeEach(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterHostNS = "vcluster-" + vClusterName
			})

			// createIngressClass creates an IngressClass on the host and registers cleanup.
			createIngressClass := func(ctx context.Context, name string, controller string, icLabels map[string]string) *networkingv1.IngressClass {
				GinkgoHelper()
				ic := &networkingv1.IngressClass{
					ObjectMeta: metav1.ObjectMeta{
						Name:   name,
						Labels: icLabels,
					},
					Spec: networkingv1.IngressClassSpec{
						Controller: controller,
					},
				}
				created, err := hostClient.NetworkingV1().IngressClasses().Create(ctx, ic, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					err := hostClient.NetworkingV1().IngressClasses().Delete(ctx, name, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})
				return created
			}

			It("only syncs ingressClasses matching the label selector to vcluster", func(ctx context.Context) {
				suffix := random.String(6)
				matchingName := "ic-match-" + suffix
				nonMatchingName := "ic-nomatch-" + suffix

				createIngressClass(ctx, matchingName, "k8s.io/ingress-nginx", map[string]string{"value": "one"})
				createIngressClass(ctx, nonMatchingName, "haproxy.org/ingress-controller", map[string]string{"value": "two"})

				By("waiting for the matching class to appear and the non-matching class to stay absent", func() {
					Eventually(func(g Gomega) {
						ingressClasses, err := vClusterClient.NetworkingV1().IngressClasses().List(ctx, metav1.ListOptions{})
						g.Expect(err).To(Succeed(), "failed to list ingressClasses in vcluster: %v", err)

						var foundMatch, foundNoMatch bool
						for _, ic := range ingressClasses.Items {
							switch ic.Name {
							case matchingName:
								foundMatch = true
							case nonMatchingName:
								foundNoMatch = true
							}
						}
						g.Expect(foundMatch).To(BeTrue(), "expected matching ingressClass to be synced to vcluster")
						g.Expect(foundNoMatch).To(BeFalse(), "expected non-matching ingressClass to stay absent from vcluster")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("does not sync ingresses created in vcluster using an ingressClass not available in vcluster", func(ctx context.Context) {
				suffix := random.String(6)
				nonMatchingName := "ic-ingressreject-" + suffix
				ingressName := "ingress-reject-" + suffix
				pathType := networkingv1.PathTypePrefix

				createIngressClass(ctx, nonMatchingName, "haproxy.org/ingress-controller", map[string]string{"value": "two"})

				By("creating an ingress using the non-synced ingressClass in vcluster", func() {
					_, err := vClusterClient.NetworkingV1().Ingresses("default").Create(ctx, &networkingv1.Ingress{
						ObjectMeta: metav1.ObjectMeta{
							Name:      ingressName,
							Namespace: "default",
						},
						Spec: networkingv1.IngressSpec{
							IngressClassName: &nonMatchingName,
							Rules: []networkingv1.IngressRule{
								{
									Host: "web-haproxy.local",
									IngressRuleValue: networkingv1.IngressRuleValue{
										HTTP: &networkingv1.HTTPIngressRuleValue{
											Paths: []networkingv1.HTTPIngressPath{
												{
													Path:     "/",
													PathType: &pathType,
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
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.NetworkingV1().Ingresses("default").Delete(ctx, ingressName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				By("verifying the ingress is not synced to the host", func() {
					translatedName := translate.SafeConcatName(ingressName, "x", "default", "x", vClusterName)
					Consistently(func(g Gomega) {
						_, err := hostClient.NetworkingV1().Ingresses(vClusterHostNS).Get(ctx, translatedName, metav1.GetOptions{})
						g.Expect(kerrors.IsNotFound(err)).To(BeTrue(), "ingress using non-synced ingressClass should not appear on host")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
				})

				By("waiting for a SyncWarning event on the ingress", func() {
					expectedMsg := fmt.Sprintf(
						`did not sync ingress "%s" to host because the ingress class "%s" in the host does not match the selector under 'sync.fromHost.ingressClasses.selector'`,
						ingressName, nonMatchingName,
					)
					Eventually(func(g Gomega) {
						eventList, err := vClusterClient.CoreV1().Events("default").List(ctx, metav1.ListOptions{})
						g.Expect(err).To(Succeed(), "failed to list events: %v", err)
						var found bool
						for _, event := range eventList.Items {
							if event.InvolvedObject.Kind == "Ingress" &&
								event.InvolvedObject.Name == ingressName &&
								event.Type == corev1.EventTypeWarning &&
								event.Reason == "SyncWarning" {
								g.Expect(event.Message).To(ContainSubstring(expectedMsg))
								found = true
								break
							}
						}
						g.Expect(found).To(BeTrue(), "expected SyncWarning event for ingress %s", ingressName)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})

			It("syncs ingresses created in vcluster to host when using an ingressClass synced from host", func(ctx context.Context) {
				suffix := random.String(6)
				matchingName := "ic-ingresssync-" + suffix
				ingressName := "ingress-sync-" + suffix
				pathType := networkingv1.PathTypePrefix

				createIngressClass(ctx, matchingName, "k8s.io/ingress-nginx", map[string]string{"value": "one"})

				By("waiting for the ingressClass to be synced to vcluster", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterClient.NetworkingV1().IngressClasses().Get(ctx, matchingName, metav1.GetOptions{})
						g.Expect(err).To(Succeed(), "ingressClass %s not yet synced to vcluster: %v", matchingName, err)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				By("creating an ingress using the synced ingressClass in vcluster", func() {
					_, err := vClusterClient.NetworkingV1().Ingresses("default").Create(ctx, &networkingv1.Ingress{
						ObjectMeta: metav1.ObjectMeta{
							Name:      ingressName,
							Namespace: "default",
						},
						Spec: networkingv1.IngressSpec{
							IngressClassName: &matchingName,
							Rules: []networkingv1.IngressRule{
								{
									Host: "web-nginx.local",
									IngressRuleValue: networkingv1.IngressRuleValue{
										HTTP: &networkingv1.HTTPIngressRuleValue{
											Paths: []networkingv1.HTTPIngressPath{
												{
													Path:     "/",
													PathType: &pathType,
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
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.NetworkingV1().Ingresses("default").Delete(ctx, ingressName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				By("waiting for the ingress to appear in the host vcluster namespace", func() {
					expectedHostIngressName := translate.SafeConcatName(ingressName, "x", "default", "x", vClusterName)
					Eventually(func(g Gomega) {
						ingresses, err := hostClient.NetworkingV1().Ingresses(vClusterHostNS).List(ctx, metav1.ListOptions{})
						g.Expect(err).To(Succeed(), "failed to list ingresses in host namespace %s: %v", vClusterHostNS, err)
						var found bool
						for _, ingress := range ingresses.Items {
							if ingress.Name == expectedHostIngressName {
								found = true
								break
							}
						}
						g.Expect(found).To(BeTrue(), "expected ingress %s to appear in host namespace %s", expectedHostIngressName, vClusterHostNS)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})
		})
}
