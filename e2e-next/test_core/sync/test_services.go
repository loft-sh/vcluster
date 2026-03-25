package test_core

import (
	"context"
	"encoding/json"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
)

// DescribeServiceBasicSync registers basic service sync tests against the given vCluster.
func DescribeServiceBasicSync(vcluster suite.Dependency) bool {
	return Describe("Service basic sync from vCluster to host",
		labels.Core,
		labels.Sync,
		labels.PR,
		cluster.Use(vcluster),
		cluster.Use(clusters.HostCluster),
		func() {
			var (
				hostClient        kubernetes.Interface
				vClusterClient    kubernetes.Interface
				vClusterClientset *kubernetes.Clientset
				vClusterConfig    *rest.Config
			)

			BeforeEach(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				vClusterConfig = cluster.CurrentClusterFrom(ctx).KubernetesRestConfig()
				Expect(vClusterConfig).NotTo(BeNil())
				var err error
				vClusterClientset, err = kubernetes.NewForConfig(vClusterConfig)
				Expect(err).To(Succeed())
			})

			It("should sync LoadBalancer service node ports and cluster IP to host", func(ctx context.Context) {
				suffix := random.String(6)
				nsName := "svc-lb-nodeport-" + suffix
				svcName := "myservice-lb-" + suffix

				By("creating the test namespace in vCluster", func() {
					_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: nsName},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				var vService *corev1.Service
				By("creating a LoadBalancer service in vCluster", func() {
					var err error
					vService, err = vClusterClient.CoreV1().Services(nsName).Create(ctx, &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      svcName,
							Namespace: nsName,
						},
						Spec: corev1.ServiceSpec{
							Type:                  corev1.ServiceTypeLoadBalancer,
							ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeLocal,
							Selector:              map[string]string{"doesnt": "matter"},
							Ports: []corev1.ServicePort{
								{Port: 80},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				pServiceName := translate.Default.HostName(nil, vService.Name, vService.Namespace)

				var pService *corev1.Service
				By("waiting for the service to be synced to the host cluster", func() {
					Eventually(func(g Gomega) {
						var err error
						pService, err = hostClient.CoreV1().Services(pServiceName.Namespace).Get(ctx, pServiceName.Name, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "host service %s/%s not yet available", pServiceName.Namespace, pServiceName.Name)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("refreshing the vCluster service to get the assigned ClusterIP and node ports", func() {
					var err error
					vService, err = vClusterClient.CoreV1().Services(nsName).Get(ctx, svcName, metav1.GetOptions{})
					Expect(err).To(Succeed())
				})

				By("asserting that ClusterIP, HealthCheckNodePort, and NodePorts match on vCluster and host", func() {
					Expect(vService.Spec.ClusterIP).To(Equal(pService.Spec.ClusterIP),
						"ClusterIP should match between vCluster and host service")
					Expect(vService.Spec.HealthCheckNodePort).To(Equal(pService.Spec.HealthCheckNodePort),
						"HealthCheckNodePort should match between vCluster and host service")
					for i := range vService.Spec.Ports {
						Expect(vService.Spec.Ports[i].NodePort).To(Equal(pService.Spec.Ports[i].NodePort),
							"NodePort at index %d should match between vCluster and host service", i)
					}
				})
			})

			It("should create a service when no Kind is present in the request body", func(ctx context.Context) {
				suffix := random.String(6)
				nsName := "svc-no-kind-" + suffix
				svcName := "myservice-" + suffix

				By("creating the test namespace in vCluster", func() {
					_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: nsName},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				service := corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      svcName,
						Namespace: nsName,
					},
					Spec: corev1.ServiceSpec{
						Selector: map[string]string{"doesnt": "matter"},
						Ports: []corev1.ServicePort{
							{Port: 80},
						},
					},
				}

				By("posting the service via raw REST with no Kind field", func() {
					body, err := json.Marshal(service)
					Expect(err).To(Succeed())
					_, err = vClusterClientset.RESTClient().Post().
						AbsPath("/api/v1/namespaces/" + nsName + "/services").
						Body(body).
						DoRaw(ctx)
					Expect(err).To(Succeed())
				})

				pServiceName := translate.Default.HostName(nil, service.Name, service.Namespace)

				By("waiting for the service to appear in vCluster", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterClient.CoreV1().Services(nsName).Get(ctx, svcName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "vCluster service %s/%s not yet available", nsName, svcName)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("waiting for the service to be synced to the host cluster", func() {
					Eventually(func(g Gomega) {
						_, err := hostClient.CoreV1().Services(pServiceName.Namespace).Get(ctx, pServiceName.Name, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "host service %s/%s not yet available", pServiceName.Namespace, pServiceName.Name)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})

			It("should complete a service status lifecycle: create, patch status, update status, patch service, delete", func(ctx context.Context) {
				suffix := random.String(6)
				nsName := "svc-status-lifecycle-" + suffix
				svcName := "test-svc-" + suffix
				svcLabels := map[string]string{"test-service-static-" + suffix: "true"}

				By("creating the test namespace in vCluster", func() {
					_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: nsName},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				By("creating a LoadBalancer service", func() {
					_, err := vClusterClient.CoreV1().Services(nsName).Create(ctx, &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:   svcName,
							Labels: svcLabels,
						},
						Spec: corev1.ServiceSpec{
							Type: corev1.ServiceTypeLoadBalancer,
							Ports: []corev1.ServicePort{{
								Name:       "http",
								Protocol:   corev1.ProtocolTCP,
								Port:       int32(80),
								TargetPort: intstr.FromInt32(80),
							}},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				svcResource := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
				dynamicClient, err := dynamic.NewForConfig(vClusterConfig)
				Expect(err).To(Succeed(), "failed to create dynamic client")

				By("waiting for the service to appear in vCluster", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterClient.CoreV1().Services(nsName).Get(ctx, svcName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "service %s/%s not yet available", nsName, svcName)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("getting /status via dynamic client", func() {
					svcStatusUnstructured, err := dynamicClient.Resource(svcResource).Namespace(nsName).Get(ctx, svcName, metav1.GetOptions{}, "status")
					Expect(err).To(Succeed(), "failed to get service status for %s/%s", nsName, svcName)
					svcStatusBytes, err := json.Marshal(svcStatusUnstructured)
					Expect(err).To(Succeed())
					var svcStatus corev1.Service
					Expect(json.Unmarshal(svcStatusBytes, &svcStatus)).To(Succeed())
				})

				By("patching the ServiceStatus with a LoadBalancer ingress IP", func() {
					lbStatus := corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{{IP: "203.0.113.1"}},
					}
					lbStatusJSON, err := json.Marshal(lbStatus)
					Expect(err).To(Succeed())
					_, err = vClusterClient.CoreV1().Services(nsName).Patch(ctx, svcName, types.MergePatchType,
						[]byte(`{"metadata":{"annotations":{"patchedstatus":"true"}},"status":{"loadBalancer":`+string(lbStatusJSON)+`}}`),
						metav1.PatchOptions{}, "status")
					Expect(err).To(Succeed(), "could not patch service status for %s/%s", nsName, svcName)
				})

				By("waiting for the patched status annotation to appear on the service", func() {
					Eventually(func(g Gomega) {
						svc, err := vClusterClient.CoreV1().Services(nsName).Get(ctx, svcName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "service %s/%s not yet available", nsName, svcName)
						g.Expect(svc.Annotations).To(HaveKeyWithValue("patchedstatus", "true"),
							"expected patchedstatus annotation on service %s/%s, got annotations: %v", nsName, svcName, svc.Annotations)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("updating the ServiceStatus to add a condition", func() {
					svcClient := vClusterClient.CoreV1().Services(nsName)
					err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
						statusToUpdate, err := svcClient.Get(ctx, svcName, metav1.GetOptions{})
						if err != nil {
							return err
						}
						statusToUpdate.Status.Conditions = append(statusToUpdate.Status.Conditions, metav1.Condition{
							Type:    "StatusUpdate",
							Status:  metav1.ConditionTrue,
							Reason:  "E2E",
							Message: "Set from e2e test",
						})
						_, err = svcClient.UpdateStatus(ctx, statusToUpdate, metav1.UpdateOptions{})
						return err
					})
					Expect(err).To(Succeed(), "failed to UpdateStatus for service %s/%s", nsName, svcName)
				})

				By("waiting for the StatusUpdate condition to appear on the service", func() {
					Eventually(func(g Gomega) {
						svc, err := vClusterClient.CoreV1().Services(nsName).Get(ctx, svcName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "service %s/%s not yet available", nsName, svcName)
						var found bool
						for _, cond := range svc.Status.Conditions {
							if cond.Type == "StatusUpdate" && cond.Reason == "E2E" && cond.Message == "Set from e2e test" {
								found = true
								break
							}
						}
						g.Expect(found).To(BeTrue(), "expected StatusUpdate condition on service %s/%s, got conditions: %v", nsName, svcName, svc.Status.Conditions)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("patching the service labels", func() {
					servicePatchPayload, err := json.Marshal(corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"test-service-" + suffix: "patched",
							},
						},
					})
					Expect(err).To(Succeed())
					_, err = vClusterClient.CoreV1().Services(nsName).Patch(ctx, svcName, types.StrategicMergePatchType, servicePatchPayload, metav1.PatchOptions{})
					Expect(err).To(Succeed(), "failed to patch service %s/%s", nsName, svcName)
				})

				By("waiting for the patched label to appear on the service", func() {
					Eventually(func(g Gomega) {
						svc, err := vClusterClient.CoreV1().Services(nsName).Get(ctx, svcName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "service %s/%s not yet available", nsName, svcName)
						g.Expect(svc.Labels).To(HaveKeyWithValue("test-service-"+suffix, "patched"),
							"expected patched label on service %s/%s, got labels: %v", nsName, svcName, svc.Labels)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("deleting the service", func() {
					err := vClusterClient.CoreV1().Services(nsName).Delete(ctx, svcName, metav1.DeleteOptions{})
					Expect(err).To(Succeed(), "failed to delete service %s/%s", nsName, svcName)
				})

				By("waiting for the service to disappear from vCluster", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterClient.CoreV1().Services(nsName).Get(ctx, svcName, metav1.GetOptions{})
						g.Expect(kerrors.IsNotFound(err)).To(BeTrue(),
							"service %s/%s should be deleted, got err: %v", nsName, svcName, err)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})

			It("should sync labels and annotations bidirectionally between vCluster and host", func(ctx context.Context) {
				suffix := random.String(6)
				nsName := "svc-bidir-sync-" + suffix
				svcName := "myservice-bidir-" + suffix

				By("creating the test namespace in vCluster", func() {
					_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: nsName},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				var vService *corev1.Service
				By("creating a headless ClusterIP service with an annotation in vCluster", func() {
					var err error
					vService, err = vClusterClient.CoreV1().Services(nsName).Create(ctx, &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      svcName,
							Namespace: nsName,
							Annotations: map[string]string{
								"some-annotation": "that is set from the vCluster",
							},
						},
						Spec: corev1.ServiceSpec{
							Type:      corev1.ServiceTypeClusterIP,
							ClusterIP: "None",
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				pServiceName := translate.Default.HostName(nil, vService.Name, vService.Namespace)

				By("waiting for the service to be synced to the host cluster", func() {
					Eventually(func(g Gomega) {
						_, err := hostClient.CoreV1().Services(pServiceName.Namespace).Get(ctx, pServiceName.Name, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "host service %s/%s not yet available", pServiceName.Namespace, pServiceName.Name)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("updating the host service to add an annotation suffix and a label", func() {
					Eventually(func(g Gomega) {
						pService, err := hostClient.CoreV1().Services(pServiceName.Namespace).Get(ctx, pServiceName.Name, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "host service %s/%s not yet available", pServiceName.Namespace, pServiceName.Name)

						if pService.Annotations == nil {
							pService.Annotations = map[string]string{}
						}
						pService.Annotations["some-annotation"] += " and update from the host cluster"

						if pService.Labels == nil {
							pService.Labels = map[string]string{}
						}
						pService.Labels["host-cluster-label"] = "some_host_label_value"

						_, err = hostClient.CoreV1().Services(pServiceName.Namespace).Update(ctx, pService, metav1.UpdateOptions{})
						// Any error (including conflict) causes Eventually to retry.
						g.Expect(err).NotTo(HaveOccurred(), "failed to update host service %s/%s", pServiceName.Namespace, pServiceName.Name)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("waiting for the host-cluster annotation and label to be synced back into the vCluster service", func() {
					Eventually(func(g Gomega) {
						updatedVService, err := vClusterClient.CoreV1().Services(nsName).Get(ctx, svcName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "vCluster service %s/%s not yet available", nsName, svcName)
						updatedPService, err := hostClient.CoreV1().Services(pServiceName.Namespace).Get(ctx, pServiceName.Name, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "host service %s/%s not yet available", pServiceName.Namespace, pServiceName.Name)

						g.Expect(updatedVService.Annotations["some-annotation"]).To(Equal(updatedPService.Annotations["some-annotation"]),
							"expected vService annotation 'some-annotation' (%q) to equal pService annotation (%q)",
							updatedVService.Annotations["some-annotation"], updatedPService.Annotations["some-annotation"])
						g.Expect(updatedVService.Labels["host-cluster-label"]).To(Equal(updatedPService.Labels["host-cluster-label"]),
							"expected vService label 'host-cluster-label' (%q) to equal pService label (%q)",
							updatedVService.Labels["host-cluster-label"], updatedPService.Labels["host-cluster-label"])
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("updating the vCluster service to add an annotation suffix and a vCluster-specific label", func() {
					Eventually(func(g Gomega) {
						updatedVService, err := vClusterClient.CoreV1().Services(nsName).Get(ctx, svcName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "vCluster service %s/%s not yet available", nsName, svcName)

						if updatedVService.Annotations == nil {
							updatedVService.Annotations = map[string]string{}
						}
						updatedVService.Annotations["some-annotation"] += " and another update from the vCluster"

						if updatedVService.Labels == nil {
							updatedVService.Labels = map[string]string{}
						}
						updatedVService.Labels["vcluster-label"] = "some_vcluster_value"

						_, err = vClusterClient.CoreV1().Services(nsName).Update(ctx, updatedVService, metav1.UpdateOptions{})
						// Any error (including conflict) causes Eventually to retry.
						g.Expect(err).NotTo(HaveOccurred(), "failed to update vCluster service %s/%s", nsName, svcName)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("waiting for the vCluster annotation and label to be synced to the host service", func() {
					Eventually(func(g Gomega) {
						updatedVService, err := vClusterClient.CoreV1().Services(nsName).Get(ctx, svcName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "vCluster service %s/%s not yet available", nsName, svcName)
						updatedPService, err := hostClient.CoreV1().Services(pServiceName.Namespace).Get(ctx, pServiceName.Name, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "host service %s/%s not yet available", pServiceName.Namespace, pServiceName.Name)

						g.Expect(updatedVService.Annotations["some-annotation"]).To(Equal(updatedPService.Annotations["some-annotation"]),
							"expected vService annotation 'some-annotation' (%q) to equal pService annotation (%q)",
							updatedVService.Annotations["some-annotation"], updatedPService.Annotations["some-annotation"])
						g.Expect(updatedVService.Labels["vcluster-label"]).To(Equal(updatedPService.Labels["vcluster-label"]),
							"expected vService label 'vcluster-label' (%q) to equal pService label (%q)",
							updatedVService.Labels["vcluster-label"], updatedPService.Labels["vcluster-label"])
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})
		})
}
