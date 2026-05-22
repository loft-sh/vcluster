package test_core

import (
	"context"
	"strings"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

const (
	// PodDNSNameserversNamespace is the host-side namespace that holds the
	// DNS Services referenced by sync.toHost.pods.dns.nameservers.
	PodDNSNameserversNamespace = "vcluster-dns-test"

	// PodDNSNameserversPrimaryService is the name of the primary host DNS
	// Service resolved by the first nameservers entry.
	PodDNSNameserversPrimaryService = "primary-dns"

	// PodDNSNameserversSecondaryService is the name of the secondary host
	// DNS Service resolved by the second nameservers entry.
	PodDNSNameserversSecondaryService = "secondary-dns"

	// PodDNSNameserversAltPrimaryService is an alternate Service that can
	// take over the "primary" selector value at runtime to exercise the
	// reschedule-on-selector-rebind path.
	PodDNSNameserversAltPrimaryService = "primary-dns-alt"

	// PodDNSNameserversLabelKey is the label key the nameservers entries
	// select on. Values: "primary" or "secondary".
	PodDNSNameserversLabelKey = "vcluster.com/dns-ns"

	// PodDNSNameserversPrimaryExpectedName is the unique A-record the
	// primary CoreDNS instance answers.
	PodDNSNameserversPrimaryExpectedName = "primary.dns-test.vcluster.com"

	// PodDNSNameserversPrimaryExpectedIP is the IP returned by the primary
	// CoreDNS instance for PodDNSNameserversPrimaryExpectedName.
	PodDNSNameserversPrimaryExpectedIP = "1.2.3.4"

	// PodDNSNameserversSecondaryExpectedName is the unique A-record the
	// secondary CoreDNS instance answers.
	PodDNSNameserversSecondaryExpectedName = "secondary.dns-test.vcluster.com"

	// PodDNSNameserversSecondaryExpectedIP is the IP returned by the
	// secondary CoreDNS instance for PodDNSNameserversSecondaryExpectedName.
	PodDNSNameserversSecondaryExpectedIP = "5.6.7.8"
)

// PodDNSNameserversSpec exercises sync.toHost.pods.dns.nameservers: pods
// synced to the host receive dnsConfig.nameservers built from the resolved
// host Services' ClusterIPs unless they opt out (hostNetwork) or override
// (user-supplied dnsConfig.nameservers).
func PodDNSNameserversSpec() {
	Describe("Pod DNS nameservers override",
		labels.Core,
		labels.Sync,
		labels.Pods,
		func() {
			var (
				hostClient     kubernetes.Interface
				vClusterClient kubernetes.Interface
				vClusterName   string
				hostNS         string
				primaryIP      string
				secondaryIP    string
			)

			BeforeEach(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				hostNS = "vcluster-" + vClusterName

				primarySvc, err := hostClient.CoreV1().Services(PodDNSNameserversNamespace).Get(ctx, PodDNSNameserversPrimaryService, metav1.GetOptions{})
				Expect(err).To(Succeed())
				primaryIP = primarySvc.Spec.ClusterIP
				Expect(primaryIP).NotTo(BeEmpty(), "primary DNS Service has no ClusterIP")

				secondarySvc, err := hostClient.CoreV1().Services(PodDNSNameserversNamespace).Get(ctx, PodDNSNameserversSecondaryService, metav1.GetOptions{})
				Expect(err).To(Succeed())
				secondaryIP = secondarySvc.Spec.ClusterIP
				Expect(secondaryIP).NotTo(BeEmpty(), "secondary DNS Service has no ClusterIP")
			})

			// createVPod creates a pod in the vCluster default namespace
			// with a generated name, registers cleanup, and returns the
			// created pod (whose Name is populated by the API server).
			createVPod := func(ctx context.Context, mutate func(*corev1.Pod)) *corev1.Pod {
				GinkgoHelper()
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{GenerateName: "dns-pod-", Namespace: "default"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:    "main",
								Image:   "busybox:1.28",
								Command: []string{"sh", "-c", "sleep 3600"},
							},
						},
					},
				}
				if mutate != nil {
					mutate(pod)
				}
				created, err := vClusterClient.CoreV1().Pods("default").Create(ctx, pod, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Pods("default").Delete(ctx, created.Name, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})
				return created
			}

			// getHostPod fetches the host-side pod corresponding to a virtual
			// pod by its translated name.
			getHostPod := func(ctx context.Context, vPodName string) (*corev1.Pod, error) {
				hostName := translate.SingleNamespaceHostName(vPodName, "default", vClusterName)
				return hostClient.CoreV1().Pods(hostNS).Get(ctx, hostName, metav1.GetOptions{})
			}

			// setServiceLabel patches the named host Service's label value or
			// removes the label entirely when value is empty. Uses retry on
			// conflict so concurrent reconcilers do not race the test.
			setServiceLabel := func(ctx context.Context, svcName, value string) {
				GinkgoHelper()
				err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
					svc, err := hostClient.CoreV1().Services(PodDNSNameserversNamespace).Get(ctx, svcName, metav1.GetOptions{})
					if err != nil {
						return err
					}
					if svc.Labels == nil {
						svc.Labels = map[string]string{}
					}
					if value == "" {
						delete(svc.Labels, PodDNSNameserversLabelKey)
					} else {
						svc.Labels[PodDNSNameserversLabelKey] = value
					}
					_, err = hostClient.CoreV1().Services(PodDNSNameserversNamespace).Update(ctx, svc, metav1.UpdateOptions{})
					return err
				})
				Expect(err).To(Succeed())
			}

			It("writes all resolved ClusterIPs to synced pods", labels.PR, func(ctx context.Context) {
				var pod *corev1.Pod

				By("Creating a simple pod in the vCluster", func() {
					pod = createVPod(ctx, nil)
				})

				By("Waiting for the host pod to receive both resolved nameservers", func() {
					Eventually(func(g Gomega) {
						hostPod, err := getHostPod(ctx, pod.Name)
						g.Expect(err).To(Succeed(), "host pod for %s not yet present", pod.Name)
						g.Expect(hostPod.Spec.DNSPolicy).To(Equal(corev1.DNSNone),
							"host pod dnsPolicy is %s, expected None", hostPod.Spec.DNSPolicy)
						g.Expect(hostPod.Spec.DNSConfig).NotTo(BeNil(), "host pod dnsConfig is nil")
						g.Expect(hostPod.Spec.DNSConfig.Nameservers).To(Equal([]string{primaryIP, secondaryIP}),
							"host pod nameservers %v != expected [%s %s]",
							hostPod.Spec.DNSConfig.Nameservers, primaryIP, secondaryIP)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Waiting for the virtual pod to be Running", func() {
					Eventually(func(g Gomega) {
						p, err := vClusterClient.CoreV1().Pods("default").Get(ctx, pod.Name, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						g.Expect(p.Status.Phase).To(Equal(corev1.PodRunning),
							"pod not yet running, phase=%s reason=%s message=%s",
							p.Status.Phase, p.Status.Reason, p.Status.Message)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Verifying DNS resolution via the injected nameservers", func() {
					vClusterRestConfig := cluster.CurrentClusterFrom(ctx).KubernetesRestConfig()
					Expect(vClusterRestConfig).NotTo(BeNil())

					Eventually(func(g Gomega) {
						stdout, stderr, err := podhelper.ExecBuffered(
							ctx, vClusterRestConfig, "default", pod.Name, "main",
							[]string{"nslookup", PodDNSNameserversPrimaryExpectedName}, nil,
						)
						g.Expect(err).NotTo(HaveOccurred(),
							"nslookup primary failed: stdout=%s stderr=%s", string(stdout), string(stderr))
						g.Expect(string(stdout)).To(ContainSubstring(PodDNSNameserversPrimaryExpectedIP),
							"primary nslookup output missing expected IP: %s", string(stdout))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

					Eventually(func(g Gomega) {
						stdout, stderr, err := podhelper.ExecBuffered(
							ctx, vClusterRestConfig, "default", pod.Name, "main",
							[]string{"nslookup", PodDNSNameserversSecondaryExpectedName}, nil,
						)
						g.Expect(err).NotTo(HaveOccurred(),
							"nslookup secondary failed: stdout=%s stderr=%s", string(stdout), string(stderr))
						g.Expect(string(stdout)).To(ContainSubstring(PodDNSNameserversSecondaryExpectedIP),
							"secondary nslookup output missing expected IP: %s", string(stdout))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})

			It("skips hostNetwork pods", func(ctx context.Context) {
				pod := createVPod(ctx, func(p *corev1.Pod) {
					p.Spec.HostNetwork = true
				})

				By("Waiting for the host pod to appear without injected nameservers", func() {
					Eventually(func(g Gomega) {
						hostPod, err := getHostPod(ctx, pod.Name)
						g.Expect(err).To(Succeed(), "host pod for %s not yet present", pod.Name)
						if hostPod.Spec.DNSConfig != nil {
							g.Expect(hostPod.Spec.DNSConfig.Nameservers).To(BeEmpty(),
								"hostNetwork pod should not have injected nameservers, got %v",
								hostPod.Spec.DNSConfig.Nameservers)
						}
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Asserting the host pod's nameservers stay empty over the polling window", func() {
					Consistently(func(g Gomega) {
						hostPod, err := getHostPod(ctx, pod.Name)
						g.Expect(err).To(Succeed(), "host pod for %s vanished", pod.Name)
						if hostPod.Spec.DNSConfig != nil {
							g.Expect(hostPod.Spec.DNSConfig.Nameservers).To(BeEmpty(),
								"hostNetwork pod gained injected nameservers, got %v",
								hostPod.Spec.DNSConfig.Nameservers)
						}
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("respects user-defined dnsConfig.nameservers", func(ctx context.Context) {
				userIP := "8.8.8.8"

				pod := createVPod(ctx, func(p *corev1.Pod) {
					p.Spec.DNSPolicy = corev1.DNSNone
					p.Spec.DNSConfig = &corev1.PodDNSConfig{
						Nameservers: []string{userIP},
					}
				})

				By("Waiting for the host pod and verifying the user nameservers are preserved", func() {
					Eventually(func(g Gomega) {
						hostPod, err := getHostPod(ctx, pod.Name)
						g.Expect(err).To(Succeed(), "host pod for %s not yet present", pod.Name)
						g.Expect(hostPod.Spec.DNSConfig).NotTo(BeNil(), "host pod dnsConfig is nil")
						g.Expect(hostPod.Spec.DNSConfig.Nameservers).To(Equal([]string{userIP}),
							"user-supplied nameservers were modified, got %v", hostPod.Spec.DNSConfig.Nameservers)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Asserting user-supplied nameservers remain stable over the polling window", func() {
					Consistently(func(g Gomega) {
						hostPod, err := getHostPod(ctx, pod.Name)
						g.Expect(err).To(Succeed(), "host pod for %s vanished", pod.Name)
						g.Expect(hostPod.Spec.DNSConfig).NotTo(BeNil(), "host pod dnsConfig is nil")
						g.Expect(hostPod.Spec.DNSConfig.Nameservers).To(Equal([]string{userIP}),
							"user-supplied nameservers were overwritten, got %v", hostPod.Spec.DNSConfig.Nameservers)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("resolves partial list when one Service is missing", func(ctx context.Context) {
				By("Removing the dns-ns label from the secondary Service", func() {
					setServiceLabel(ctx, PodDNSNameserversSecondaryService, "")
					DeferCleanup(func(ctx context.Context) {
						setServiceLabel(ctx, PodDNSNameserversSecondaryService, "secondary")
					})
				})

				pod := createVPod(ctx, nil)

				By("Waiting for the host pod to receive only the primary nameserver", func() {
					Eventually(func(g Gomega) {
						hostPod, err := getHostPod(ctx, pod.Name)
						g.Expect(err).To(Succeed(), "host pod for %s not yet present", pod.Name)
						g.Expect(hostPod.Spec.DNSConfig).NotTo(BeNil(), "host pod dnsConfig is nil")
						g.Expect(hostPod.Spec.DNSConfig.Nameservers).To(Equal([]string{primaryIP}),
							"expected only primary nameserver, got %v", hostPod.Spec.DNSConfig.Nameservers)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Waiting for a warning event on the virtual pod about the unresolved nameserver", func() {
					Eventually(func(g Gomega) {
						events, err := vClusterClient.CoreV1().Events("default").List(ctx, metav1.ListOptions{
							FieldSelector: "involvedObject.name=" + pod.Name,
						})
						g.Expect(err).To(Succeed(), "failed to list events for pod %s", pod.Name)
						var found bool
						for _, ev := range events.Items {
							if ev.Type == corev1.EventTypeWarning &&
								(containsAny(ev.Message, "nameserver", "DNS") || containsAny(ev.Reason, "DNS", "Nameserver")) {
								found = true
								break
							}
						}
						g.Expect(found).To(BeTrue(),
							"no DNS nameserver warning event found for pod %s, events: %d",
							pod.Name, len(events.Items))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})

			It("blocks pod creation when all Services are missing", func(ctx context.Context) {
				By("Removing the dns-ns label from both Services", func() {
					setServiceLabel(ctx, PodDNSNameserversPrimaryService, "")
					DeferCleanup(func(ctx context.Context) {
						setServiceLabel(ctx, PodDNSNameserversPrimaryService, "primary")
					})
					setServiceLabel(ctx, PodDNSNameserversSecondaryService, "")
					DeferCleanup(func(ctx context.Context) {
						setServiceLabel(ctx, PodDNSNameserversSecondaryService, "secondary")
					})
				})

				pod := createVPod(ctx, nil)

				By("Verifying the host pod is not created", func() {
					Consistently(func(g Gomega) {
						_, err := getHostPod(ctx, pod.Name)
						g.Expect(kerrors.IsNotFound(err)).To(BeTrue(),
							"host pod for %s should not exist, got err=%v", pod.Name, err)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
				})
			})

			It("re-resolves nameservers when a selected Service's ClusterIP changes", func(ctx context.Context) {
				// Use a Deployment so a ReplicaSet replaces the pod after the
				// syncer deletes it. Standalone pods would not be recreated
				// because the syncer treats a missing host pod with a
				// non-empty StartTime as terminal and deletes the virtual pod.
				deployName := "dns-dep-rebind-ip"
				selector := map[string]string{"app": deployName}
				replicas := int32(1)

				By("Creating a single-replica Deployment in the vCluster", func() {
					_, err := vClusterClient.AppsV1().Deployments("default").Create(ctx, &appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{Name: deployName, Namespace: "default"},
						Spec: appsv1.DeploymentSpec{
							Replicas: &replicas,
							Selector: &metav1.LabelSelector{MatchLabels: selector},
							Template: corev1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{Labels: selector},
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{{
										Name:    "main",
										Image:   "busybox:1.28",
										Command: []string{"sh", "-c", "sleep 3600"},
									}},
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := vClusterClient.AppsV1().Deployments("default").Delete(ctx, deployName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).To(Succeed())
						}
					})
				})

				findDeploymentPod := func(ctx context.Context) (*corev1.Pod, error) {
					pods, err := vClusterClient.CoreV1().Pods("default").List(ctx, metav1.ListOptions{
						LabelSelector: "app=" + deployName,
					})
					if err != nil {
						return nil, err
					}
					for i := range pods.Items {
						if pods.Items[i].DeletionTimestamp == nil {
							return &pods.Items[i], nil
						}
					}
					return nil, kerrors.NewNotFound(corev1.Resource("pods"), deployName)
				}

				var oldPodName, oldHostPodUID string
				By("Waiting for the Deployment's pod to receive the original primary ClusterIP", func() {
					Eventually(func(g Gomega) {
						vPod, err := findDeploymentPod(ctx)
						g.Expect(err).To(Succeed())
						hostPod, err := getHostPod(ctx, vPod.Name)
						g.Expect(err).To(Succeed(), "host pod for %s not yet present", vPod.Name)
						g.Expect(hostPod.Spec.DNSConfig).NotTo(BeNil(), "host pod dnsConfig is nil")
						g.Expect(hostPod.Spec.DNSConfig.Nameservers).To(ContainElement(primaryIP),
							"host pod nameservers %v missing original primary IP %s",
							hostPod.Spec.DNSConfig.Nameservers, primaryIP)
						oldPodName = vPod.Name
						oldHostPodUID = string(hostPod.UID)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				var newPrimaryIP string
				By("Deleting and recreating primary-dns to force a new ClusterIP", func() {
					origSvc, err := hostClient.CoreV1().Services(PodDNSNameserversNamespace).Get(ctx, PodDNSNameserversPrimaryService, metav1.GetOptions{})
					Expect(err).To(Succeed())

					err = hostClient.CoreV1().Services(PodDNSNameserversNamespace).Delete(ctx, PodDNSNameserversPrimaryService, metav1.DeleteOptions{})
					Expect(err).To(Succeed())

					Eventually(func(g Gomega) {
						_, err := hostClient.CoreV1().Services(PodDNSNameserversNamespace).Get(ctx, PodDNSNameserversPrimaryService, metav1.GetOptions{})
						g.Expect(kerrors.IsNotFound(err)).To(BeTrue(),
							"primary-dns not yet deleted, err=%v", err)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

					recreated := &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      PodDNSNameserversPrimaryService,
							Namespace: PodDNSNameserversNamespace,
							Labels:    origSvc.Labels,
						},
						Spec: corev1.ServiceSpec{
							Type:     corev1.ServiceTypeClusterIP,
							Selector: origSvc.Spec.Selector,
							Ports:    stripPortNodePort(origSvc.Spec.Ports),
						},
					}

					var created *corev1.Service
					Eventually(func(g Gomega) {
						var err error
						created, err = hostClient.CoreV1().Services(PodDNSNameserversNamespace).Create(ctx, recreated, metav1.CreateOptions{})
						g.Expect(err).To(Succeed(), "recreate primary-dns failed: %v", err)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())

					Expect(created.Spec.ClusterIP).NotTo(BeEmpty(), "recreated primary-dns has no ClusterIP")
					Expect(created.Spec.ClusterIP).NotTo(Equal(primaryIP),
						"recreated primary-dns reused old ClusterIP %s", primaryIP)
					newPrimaryIP = created.Spec.ClusterIP
				})

				By("Waiting for the Deployment's pod to be replaced with the new primary IP", func() {
					Eventually(func(g Gomega) {
						vPod, err := findDeploymentPod(ctx)
						g.Expect(err).To(Succeed())
						g.Expect(vPod.Name).NotTo(Equal(oldPodName),
							"Deployment pod %s was not replaced", oldPodName)
						hostPod, err := getHostPod(ctx, vPod.Name)
						g.Expect(err).To(Succeed(), "host pod for %s not yet present", vPod.Name)
						g.Expect(string(hostPod.UID)).NotTo(Equal(oldHostPodUID),
							"host pod was not recreated, UID %s unchanged", oldHostPodUID)
						g.Expect(hostPod.Spec.DNSConfig).NotTo(BeNil(), "host pod dnsConfig is nil")
						g.Expect(hostPod.Spec.DNSConfig.Nameservers).To(ContainElement(newPrimaryIP),
							"host pod nameservers %v missing new primary IP %s",
							hostPod.Spec.DNSConfig.Nameservers, newPrimaryIP)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Asserting the new primary IP remains stable over the polling window", func() {
					Consistently(func(g Gomega) {
						vPod, err := findDeploymentPod(ctx)
						g.Expect(err).To(Succeed())
						hostPod, err := getHostPod(ctx, vPod.Name)
						g.Expect(err).To(Succeed(), "host pod for %s vanished", vPod.Name)
						g.Expect(hostPod.Spec.DNSConfig).NotTo(BeNil(), "host pod dnsConfig is nil")
						g.Expect(hostPod.Spec.DNSConfig.Nameservers).To(ContainElement(newPrimaryIP),
							"host pod nameservers %v lost new primary IP %s",
							hostPod.Spec.DNSConfig.Nameservers, newPrimaryIP)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("re-resolves nameservers when the label selector matches a different Service", func(ctx context.Context) {
				By("Creating an alternate primary Service without the selector label", func() {
					altSvc := &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      PodDNSNameserversAltPrimaryService,
							Namespace: PodDNSNameserversNamespace,
						},
						Spec: corev1.ServiceSpec{
							Type: corev1.ServiceTypeClusterIP,
							Ports: []corev1.ServicePort{
								{Name: "dns", Port: 53, Protocol: corev1.ProtocolUDP},
								{Name: "dns-tcp", Port: 53, Protocol: corev1.ProtocolTCP},
							},
						},
					}
					_, err := hostClient.CoreV1().Services(PodDNSNameserversNamespace).Create(ctx, altSvc, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := hostClient.CoreV1().Services(PodDNSNameserversNamespace).Delete(ctx, PodDNSNameserversAltPrimaryService, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).To(Succeed())
						}
					})
				})

				By("Moving the primary selector value from primary-dns to primary-dns-alt", func() {
					setServiceLabel(ctx, PodDNSNameserversPrimaryService, "")
					DeferCleanup(func(ctx context.Context) {
						setServiceLabel(ctx, PodDNSNameserversPrimaryService, "primary")
					})
					setServiceLabel(ctx, PodDNSNameserversAltPrimaryService, "primary")
				})

				altSvc, err := hostClient.CoreV1().Services(PodDNSNameserversNamespace).Get(ctx, PodDNSNameserversAltPrimaryService, metav1.GetOptions{})
				Expect(err).To(Succeed())
				altIP := altSvc.Spec.ClusterIP
				Expect(altIP).NotTo(BeEmpty(), "primary-dns-alt has no ClusterIP")
				Expect(altIP).NotTo(Equal(primaryIP), "primary-dns-alt reused primary-dns ClusterIP")

				pod := createVPod(ctx, nil)

				By("Waiting for the host pod to be created with the alternate primary IP", func() {
					Eventually(func(g Gomega) {
						hostPod, err := getHostPod(ctx, pod.Name)
						g.Expect(err).To(Succeed(), "host pod for %s not yet present", pod.Name)
						g.Expect(hostPod.Spec.DNSConfig).NotTo(BeNil(), "host pod dnsConfig is nil")
						g.Expect(hostPod.Spec.DNSConfig.Nameservers).To(ContainElement(altIP),
							"host pod nameservers %v missing alt primary IP %s",
							hostPod.Spec.DNSConfig.Nameservers, altIP)
						g.Expect(hostPod.Spec.DNSConfig.Nameservers).NotTo(ContainElement(primaryIP),
							"host pod nameservers %v still contains original primary IP %s",
							hostPod.Spec.DNSConfig.Nameservers, primaryIP)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Asserting the alt primary IP remains stable over the polling window", func() {
					Consistently(func(g Gomega) {
						hostPod, err := getHostPod(ctx, pod.Name)
						g.Expect(err).To(Succeed(), "host pod for %s vanished", pod.Name)
						g.Expect(hostPod.Spec.DNSConfig).NotTo(BeNil(), "host pod dnsConfig is nil")
						g.Expect(hostPod.Spec.DNSConfig.Nameservers).To(ContainElement(altIP),
							"host pod nameservers %v lost alt primary IP %s",
							hostPod.Spec.DNSConfig.Nameservers, altIP)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})
		},
	)
}

// containsAny reports whether s contains any of the given substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if sub == "" {
			continue
		}
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// stripPortNodePort clears NodePort and TargetPort.IntVal on each port so
// a recreated Service does not collide on NodePort allocation while
// preserving Port and Protocol.
func stripPortNodePort(ports []corev1.ServicePort) []corev1.ServicePort {
	out := make([]corev1.ServicePort, len(ports))
	for i, p := range ports {
		out[i] = corev1.ServicePort{
			Name:       p.Name,
			Port:       p.Port,
			Protocol:   p.Protocol,
			TargetPort: p.TargetPort,
		}
	}
	return out
}
