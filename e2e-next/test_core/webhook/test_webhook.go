package webhook

import (
	"context"
	"fmt"
	"strings"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
)

const (
	webhookImage = "registry.k8s.io/e2e-test-images/agnhost:2.33"
	pauseImage   = "registry.k8s.io/pause:3.6"
)

// Webhook tests require deploying a server, certs, and RBAC - this is a legitimate
// use of Ordered because specs depend on the webhook infrastructure created in BeforeAll.
var _ = Describe("AdmissionWebhook",
	Ordered,
	labels.Core,
	labels.Security,
	labels.Webhooks,
	cluster.Use(clusters.K8sDefaultEndpointVCluster),
	cluster.Use(clusters.HostCluster),
	func() {
		var (
			vClusterClient kubernetes.Interface
			ns             string
			certCtx        *certContext
			uniqueName     string
			servicePort    = int32(8443)
			containerPort  = int32(8444)
			svcName        = "e2e-test-webhook"
			deployName     = "sample-webhook-deployment"
			secretName     = "sample-webhook-secret"
			rbName         = "webhook-auth-reader"
		)

		BeforeAll(func(ctx context.Context) {
			vClusterClient = cluster.CurrentKubeClientFrom(ctx)
			Expect(vClusterClient).NotTo(BeNil())

			suffix := random.String(6)
			ns = "webhook-test-" + suffix
			uniqueName = "webhook-test-" + suffix

			_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   ns,
					Labels: map[string]string{uniqueName: "true"},
				},
			}, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				err := vClusterClient.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{})
				if !kerrors.IsNotFound(err) {
					Expect(err).To(Succeed())
				}
			})

			_, err = vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   ns + "-markers",
					Labels: map[string]string{uniqueName + "-markers": "true"},
				},
			}, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				err := vClusterClient.CoreV1().Namespaces().Delete(ctx, ns+"-markers", metav1.DeleteOptions{})
				if !kerrors.IsNotFound(err) {
					Expect(err).To(Succeed())
				}
			})

			By("Setting up server certificates", func() {
				certCtx = setupServerCert(ns, svcName)
			})

			_, err = vClusterClient.RbacV1().RoleBindings("kube-system").Create(ctx, &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:        rbName,
					Annotations: map[string]string{rbacv1.AutoUpdateAnnotationKey: "true"},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "",
					Kind:     "Role",
					Name:     "extension-apiserver-authentication-reader",
				},
				Subjects: []rbacv1.Subject{
					{Kind: "ServiceAccount", Name: "default", Namespace: ns},
				},
			}, metav1.CreateOptions{})
			if err != nil && !kerrors.IsAlreadyExists(err) {
				Expect(err).To(Succeed())
			}
			DeferCleanup(func(ctx context.Context) {
				err := vClusterClient.RbacV1().RoleBindings("kube-system").Delete(ctx, rbName, metav1.DeleteOptions{})
				if !kerrors.IsNotFound(err) {
					Expect(err).To(Succeed())
				}
			})

			By("Deploying the webhook server", func() {
				_, err := vClusterClient.CoreV1().Secrets(ns).Create(ctx, &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: secretName},
					Type:       corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"tls.crt": certCtx.cert,
						"tls.key": certCtx.key,
					},
				}, metav1.CreateOptions{})
				Expect(err).To(Succeed())

				// Create the deployment
				podLabels := map[string]string{"app": "sample-webhook", "webhook": "true"}
				zero := int64(0)
				replicas := int32(1)
				_, err = vClusterClient.AppsV1().Deployments(ns).Create(ctx, &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: deployName, Labels: podLabels},
					Spec: appsv1.DeploymentSpec{
						Replicas: &replicas,
						Selector: &metav1.LabelSelector{MatchLabels: podLabels},
						Strategy: appsv1.DeploymentStrategy{Type: appsv1.RollingUpdateDeploymentStrategyType},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{Labels: podLabels},
							Spec: corev1.PodSpec{
								TerminationGracePeriodSeconds: &zero,
								Containers: []corev1.Container{
									{
										Name:  "sample-webhook",
										Image: webhookImage,
										Args: []string{
											"webhook",
											"--tls-cert-file=/webhook.local.config/certificates/tls.crt",
											"--tls-private-key-file=/webhook.local.config/certificates/tls.key",
											"--alsologtostderr",
											"-v=4",
											fmt.Sprintf("--port=%d", containerPort),
										},
										ReadinessProbe: &corev1.Probe{
											ProbeHandler: corev1.ProbeHandler{
												HTTPGet: &corev1.HTTPGetAction{
													Scheme: corev1.URISchemeHTTPS,
													Port:   intstr.FromInt(int(containerPort)),
													Path:   "/readyz",
												},
											},
											PeriodSeconds:    1,
											SuccessThreshold: 1,
											FailureThreshold: 30,
										},
										Ports: []corev1.ContainerPort{{ContainerPort: containerPort}},
										VolumeMounts: []corev1.VolumeMount{
											{Name: "webhook-certs", ReadOnly: true, MountPath: "/webhook.local.config/certificates"},
										},
									},
								},
								Volumes: []corev1.Volume{
									{
										Name: "webhook-certs",
										VolumeSource: corev1.VolumeSource{
											Secret: &corev1.SecretVolumeSource{SecretName: secretName},
										},
									},
								},
							},
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).To(Succeed())
			})

			By("Waiting for the webhook deployment to be ready", func() {
				Eventually(func(g Gomega) {
					d, err := vClusterClient.AppsV1().Deployments(ns).Get(ctx, deployName, metav1.GetOptions{})
					g.Expect(err).To(Succeed())
					g.Expect(d.Status.AvailableReplicas).To(Equal(int32(1)),
						"deployment %s has %d available replicas, expected 1", deployName, d.Status.AvailableReplicas)
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
			})

			By("Creating the webhook service", func() {
				_, err := vClusterClient.CoreV1().Services(ns).Create(ctx, &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{Name: svcName, Namespace: ns, Labels: map[string]string{"test": "webhook"}},
					Spec: corev1.ServiceSpec{
						Selector: map[string]string{"webhook": "true"},
						Ports: []corev1.ServicePort{
							{Protocol: "TCP", Port: servicePort, TargetPort: intstr.FromInt(int(containerPort))},
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).To(Succeed())
			})

			By("Waiting for the service to have a ready endpoint", func() {
				Eventually(func(g Gomega) {
					esList, err := vClusterClient.DiscoveryV1().EndpointSlices(ns).List(ctx, metav1.ListOptions{
						LabelSelector: fmt.Sprintf("kubernetes.io/service-name=%s", svcName),
					})
					g.Expect(err).To(Succeed())
					hasReady := false
					for _, es := range esList.Items {
						for _, ep := range es.Endpoints {
							if ep.Conditions.Ready != nil && *ep.Conditions.Ready {
								hasReady = true
							}
						}
					}
					g.Expect(hasReady).To(BeTrue(), "no ready endpoint found for service %s", svcName)
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
			})
		})

		// registerWebhook creates a ValidatingWebhookConfiguration and waits for it to be ready.
		registerWebhook := func(ctx context.Context, configName string) func(context.Context) {
			GinkgoHelper()
			sideEffectsNone := admissionregistrationv1.SideEffectClassNone
			policyIgnore := admissionregistrationv1.Ignore
			failOpen := admissionregistrationv1.Ignore

			_, err := vClusterClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(ctx,
				&admissionregistrationv1.ValidatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{Name: configName},
					Webhooks: []admissionregistrationv1.ValidatingWebhook{
						// Deny non-compliant pods
						{
							Name: "deny-unwanted-pod-container-name-and-label.k8s.io",
							Rules: []admissionregistrationv1.RuleWithOperations{{
								Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create},
								Rule:       admissionregistrationv1.Rule{APIGroups: []string{""}, APIVersions: []string{"v1"}, Resources: []string{"pods"}},
							}},
							ClientConfig: admissionregistrationv1.WebhookClientConfig{
								Service:  &admissionregistrationv1.ServiceReference{Namespace: ns, Name: svcName, Path: strPtr("/pods"), Port: ptr.To(servicePort)},
								CABundle: certCtx.signingCert,
							},
							SideEffects:             &sideEffectsNone,
							AdmissionReviewVersions: []string{"v1", "v1beta1"},
							NamespaceSelector:       &metav1.LabelSelector{MatchLabels: map[string]string{uniqueName: "true"}},
						},
						// Deny non-compliant configmaps
						{
							Name: "deny-unwanted-configmap-data.k8s.io",
							Rules: []admissionregistrationv1.RuleWithOperations{{
								Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update, admissionregistrationv1.Delete},
								Rule:       admissionregistrationv1.Rule{APIGroups: []string{""}, APIVersions: []string{"v1"}, Resources: []string{"configmaps"}},
							}},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels:      map[string]string{uniqueName: "true"},
								MatchExpressions: []metav1.LabelSelectorRequirement{{Key: "skip-webhook-admission", Operator: metav1.LabelSelectorOpNotIn, Values: []string{"yes"}}},
							},
							ClientConfig: admissionregistrationv1.WebhookClientConfig{
								Service:  &admissionregistrationv1.ServiceReference{Namespace: ns, Name: svcName, Path: strPtr("/configmaps"), Port: ptr.To(servicePort)},
								CABundle: certCtx.signingCert,
							},
							SideEffects:             &sideEffectsNone,
							AdmissionReviewVersions: []string{"v1", "v1beta1"},
						},
						// Fail-open webhook (server cannot reach it)
						{
							Name: "fail-open.k8s.io",
							Rules: []admissionregistrationv1.RuleWithOperations{{
								Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create},
								Rule:       admissionregistrationv1.Rule{APIGroups: []string{""}, APIVersions: []string{"v1"}, Resources: []string{"configmaps"}},
							}},
							ClientConfig: admissionregistrationv1.WebhookClientConfig{
								Service: &admissionregistrationv1.ServiceReference{Namespace: ns, Name: svcName, Path: strPtr("/configmaps"), Port: ptr.To(servicePort)},
							},
							FailurePolicy:           &policyIgnore,
							SideEffects:             &sideEffectsNone,
							AdmissionReviewVersions: []string{"v1", "v1beta1"},
							NamespaceSelector:       &metav1.LabelSelector{MatchLabels: map[string]string{uniqueName: "true"}},
						},
						// Ready marker webhook
						{
							Name: "validating-is-webhook-configuration-ready.k8s.io",
							Rules: []admissionregistrationv1.RuleWithOperations{{
								Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create},
								Rule:       admissionregistrationv1.Rule{APIGroups: []string{""}, APIVersions: []string{"v1"}, Resources: []string{"configmaps"}},
							}},
							ClientConfig: admissionregistrationv1.WebhookClientConfig{
								Service:  &admissionregistrationv1.ServiceReference{Namespace: ns, Name: svcName, Path: strPtr("/always-deny"), Port: ptr.To(servicePort)},
								CABundle: certCtx.signingCert,
							},
							FailurePolicy:           &failOpen,
							SideEffects:             &sideEffectsNone,
							AdmissionReviewVersions: []string{"v1", "v1beta1"},
							NamespaceSelector:       &metav1.LabelSelector{MatchLabels: map[string]string{uniqueName + "-markers": "true"}},
							ObjectSelector:          &metav1.LabelSelector{MatchLabels: map[string]string{uniqueName: "true"}},
						},
					},
				}, metav1.CreateOptions{})
			Expect(err).To(Succeed())

			// Wait for webhook to be ready by sending marker requests
			Eventually(func(g Gomega) {
				marker := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "marker-",
						Labels:       map[string]string{uniqueName: "true"},
					},
				}
				_, err := vClusterClient.CoreV1().ConfigMaps(ns+"-markers").Create(ctx, marker, metav1.CreateOptions{})
				if err != nil && strings.Contains(err.Error(), "denied") {
					return // webhook is ready - it denied the marker
				}
				// If created, the webhook isn't intercepting yet - clean up and retry
				if err == nil {
					_ = vClusterClient.CoreV1().ConfigMaps(ns+"-markers").Delete(ctx, marker.Name, metav1.DeleteOptions{})
				}
				g.Expect(true).To(BeFalse(), "webhook not yet ready")
			}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

			return func(ctx context.Context) {
				err := vClusterClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(ctx, configName, metav1.DeleteOptions{})
				if !kerrors.IsNotFound(err) {
					Expect(err).To(Succeed())
				}
			}
		}

		// Spec 1 depends on BeforeAll's webhook infrastructure
		It("should be able to deny pod and configmap creation", func(ctx context.Context) {
			webhookCleanup := registerWebhook(ctx, "deny-pod-and-configmap-creation-"+random.String(6))
			DeferCleanup(webhookCleanup)

			By("Attempting to create a non-compliant pod (should be denied)", func() {
				_, err := vClusterClient.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "disallowed-pod",
						Labels: map[string]string{"webhook-e2e-test": "webhook-disallow"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "webhook-disallow", Image: pauseImage}},
					},
				}, metav1.CreateOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("the pod contains unwanted container name"))
				Expect(err.Error()).To(ContainSubstring("the pod contains unwanted label"))
			})

			By("Attempting to create a hanging pod (should timeout)", func() {
				_, err := vClusterClient.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "hanging-pod",
						Labels: map[string]string{"webhook-e2e-test": "wait-forever"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "wait-forever", Image: pauseImage}},
					},
				}, metav1.CreateOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("webhook"))
				Expect(err.Error()).To(ContainSubstring("deadline"))

				// Verify the pod was not created
				_, err = vClusterClient.CoreV1().Pods(ns).Get(ctx, "hanging-pod", metav1.GetOptions{})
				Expect(kerrors.IsNotFound(err)).To(BeTrue())
			})

			By("Attempting to create a non-compliant configmap (should be denied)", func() {
				_, err := vClusterClient.CoreV1().ConfigMaps(ns).Create(ctx, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "disallowed-configmap"},
					Data:       map[string]string{"webhook-e2e-test": "webhook-disallow"},
				}, metav1.CreateOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("the configmap contains unwanted key and value"))
			})

			allowedCMName := "allowed-configmap-" + random.String(6)

			By("Creating a compliant configmap (should succeed)", func() {
				_, err := vClusterClient.CoreV1().ConfigMaps(ns).Create(ctx, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: allowedCMName},
					Data:       map[string]string{"admit": "this"},
				}, metav1.CreateOptions{})
				Expect(err).To(Succeed())
			})

			By("Updating the compliant configmap to non-compliant via PUT (should be denied)", func() {
				err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
					cm, err := vClusterClient.CoreV1().ConfigMaps(ns).Get(ctx, allowedCMName, metav1.GetOptions{})
					if err != nil {
						return err
					}
					if cm.Data == nil {
						cm.Data = map[string]string{}
					}
					cm.Data["webhook-e2e-test"] = "webhook-disallow"
					_, err = vClusterClient.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{})
					return err
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("the configmap contains unwanted key and value"))
			})

			By("Patching the compliant configmap to non-compliant (should be denied)", func() {
				_, err := vClusterClient.CoreV1().ConfigMaps(ns).Patch(ctx, allowedCMName,
					types.StrategicMergePatchType,
					[]byte(`{"data":{"webhook-e2e-test":"webhook-disallow"}}`),
					metav1.PatchOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("the configmap contains unwanted key and value"))
			})

			By("Creating a non-compliant configmap in a whitelisted namespace (should succeed)", func() {
				skipNS := "exempted-ns-" + random.String(6)
				_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: skipNS,
						Labels: map[string]string{
							"skip-webhook-admission": "yes",
							uniqueName:               "true",
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Namespaces().Delete(ctx, skipNS, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				_, err = vClusterClient.CoreV1().ConfigMaps(skipNS).Create(ctx, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "disallowed-configmap"},
					Data:       map[string]string{"webhook-e2e-test": "webhook-disallow"},
				}, metav1.CreateOptions{})
				Expect(err).To(Succeed())
			})
		})

		// Spec 2 depends on BeforeAll's webhook infrastructure
		It("should be able to deny attaching to a pod", func(ctx context.Context) {
			sideEffectsNone := admissionregistrationv1.SideEffectClassNone
			failOpen := admissionregistrationv1.Ignore
			configName := "deny-attaching-to-pod-" + random.String(6)

			By("Registering the attach-deny webhook", func() {
				_, err := vClusterClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(ctx,
					&admissionregistrationv1.ValidatingWebhookConfiguration{
						ObjectMeta: metav1.ObjectMeta{Name: configName},
						Webhooks: []admissionregistrationv1.ValidatingWebhook{
							{
								Name: "deny-attaching-pod.k8s.io",
								Rules: []admissionregistrationv1.RuleWithOperations{{
									Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Connect},
									Rule:       admissionregistrationv1.Rule{APIGroups: []string{""}, APIVersions: []string{"v1"}, Resources: []string{"pods/attach"}},
								}},
								ClientConfig: admissionregistrationv1.WebhookClientConfig{
									Service:  &admissionregistrationv1.ServiceReference{Namespace: ns, Name: svcName, Path: strPtr("/pods/attach"), Port: ptr.To(servicePort)},
									CABundle: certCtx.signingCert,
								},
								SideEffects:             &sideEffectsNone,
								AdmissionReviewVersions: []string{"v1", "v1beta1"},
								NamespaceSelector:       &metav1.LabelSelector{MatchLabels: map[string]string{uniqueName: "true"}},
							},
							// Ready marker webhook
							{
								Name: "validating-is-webhook-configuration-ready.k8s.io",
								Rules: []admissionregistrationv1.RuleWithOperations{{
									Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create},
									Rule:       admissionregistrationv1.Rule{APIGroups: []string{""}, APIVersions: []string{"v1"}, Resources: []string{"configmaps"}},
								}},
								ClientConfig: admissionregistrationv1.WebhookClientConfig{
									Service:  &admissionregistrationv1.ServiceReference{Namespace: ns, Name: svcName, Path: strPtr("/always-deny"), Port: ptr.To(servicePort)},
									CABundle: certCtx.signingCert,
								},
								FailurePolicy:           &failOpen,
								SideEffects:             &sideEffectsNone,
								AdmissionReviewVersions: []string{"v1", "v1beta1"},
								NamespaceSelector:       &metav1.LabelSelector{MatchLabels: map[string]string{uniqueName + "-markers": "true"}},
								ObjectSelector:          &metav1.LabelSelector{MatchLabels: map[string]string{uniqueName: "true"}},
							},
						},
					}, metav1.CreateOptions{})
				Expect(err).To(Succeed())
			})
			DeferCleanup(func(ctx context.Context) {
				err := vClusterClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(ctx, configName, metav1.DeleteOptions{})
				if !kerrors.IsNotFound(err) {
					Expect(err).To(Succeed())
				}
			})

			By("Waiting for the attach-deny webhook to be ready", func() {
				Eventually(func(g Gomega) {
					marker := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							GenerateName: "marker-",
							Labels:       map[string]string{uniqueName: "true"},
						},
					}
					_, err := vClusterClient.CoreV1().ConfigMaps(ns+"-markers").Create(ctx, marker, metav1.CreateOptions{})
					if err != nil && strings.Contains(err.Error(), "denied") {
						return // webhook is ready
					}
					if err == nil {
						_ = vClusterClient.CoreV1().ConfigMaps(ns+"-markers").Delete(ctx, marker.Name, metav1.DeleteOptions{})
					}
					g.Expect(true).To(BeFalse(), "attach webhook not yet ready")
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})

			podName := "to-be-attached-pod-" + random.String(6)

			By("Creating a pod to attach to", func() {
				_, err := vClusterClient.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: podName},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "container1", Image: pauseImage}},
					},
				}, metav1.CreateOptions{})
				Expect(err).To(Succeed())
			})

			By("Waiting for the pod to be running", func() {
				Eventually(func(g Gomega) {
					pod, err := vClusterClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
					g.Expect(err).To(Succeed())
					g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning),
						"pod %s phase is %s, expected Running", podName, pod.Status.Phase)
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
			})

			By("Attempting to attach to the pod (should be denied)", func() {
				// Use the SubResource API to attempt an attach
				req := vClusterClient.CoreV1().RESTClient().Post().
					Resource("pods").
					Namespace(ns).
					Name(podName).
					SubResource("attach").
					Param("container", "container1")

				result := req.Do(ctx)
				err := result.Error()
				Expect(err).To(HaveOccurred(), "attach should be denied")
				Expect(err.Error()).To(ContainSubstring("attaching to pod"),
					"expected denial message about attaching to pod, got: %s", err.Error())
			})
		})
	},
)

func strPtr(s string) *string { return &s }
