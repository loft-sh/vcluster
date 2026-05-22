package test_core

import (
	"context"
	"strings"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

	// PodDNSNameserversLabelKey is the label key the nameservers entries
	// select on. Values: "primary" or "secondary".
	PodDNSNameserversLabelKey = "vcluster.loft.sh/dns-ns"

	// podDNSNameserversTenantHostLabel is the label written by the syncer
	// onto host pods carrying the resolved nameservers.
	podDNSNameserversTenantHostLabel = "vcluster.loft.sh/tenant-host-namespace"
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

			// createVPod creates a pod in the vCluster default namespace and
			// registers cleanup. Returns the pod's name.
			createVPod := func(ctx context.Context, name string, mutate func(*corev1.Pod)) {
				GinkgoHelper()
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "main", Image: "nginxinc/nginx-unprivileged:stable-alpine3.20-slim"},
						},
					},
				}
				if mutate != nil {
					mutate(pod)
				}
				_, err := vClusterClient.CoreV1().Pods("default").Create(ctx, pod, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Pods("default").Delete(ctx, name, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})
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

			It("writes all resolved ClusterIPs and tenant-identity label to synced pods", func(ctx context.Context) {
				podName := "dns-pod-default-" + random.String(6)

				By("Creating a simple pod in the vCluster", func() {
					createVPod(ctx, podName, nil)
				})

				By("Waiting for the host pod to receive both resolved nameservers and the tenant label", func() {
					Eventually(func(g Gomega) {
						hostPod, err := getHostPod(ctx, podName)
						g.Expect(err).To(Succeed(), "host pod for %s not yet present", podName)
						g.Expect(hostPod.Spec.DNSPolicy).To(Equal(corev1.DNSNone),
							"host pod dnsPolicy is %s, expected None", hostPod.Spec.DNSPolicy)
						g.Expect(hostPod.Spec.DNSConfig).NotTo(BeNil(), "host pod dnsConfig is nil")
						g.Expect(hostPod.Spec.DNSConfig.Nameservers).To(Equal([]string{primaryIP, secondaryIP}),
							"host pod nameservers %v != expected [%s %s]",
							hostPod.Spec.DNSConfig.Nameservers, primaryIP, secondaryIP)
						g.Expect(hostPod.Labels).To(HaveKey(podDNSNameserversTenantHostLabel),
							"tenant-host-namespace label missing, labels: %v", hostPod.Labels)
						g.Expect(hostPod.Labels[podDNSNameserversTenantHostLabel]).NotTo(BeEmpty(),
							"tenant-host-namespace label value is empty")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})

			It("skips hostNetwork pods", func(ctx context.Context) {
				podName := "dns-pod-hostnet-" + random.String(6)

				By("Creating a hostNetwork pod in the vCluster", func() {
					createVPod(ctx, podName, func(p *corev1.Pod) {
						p.Spec.HostNetwork = true
					})
				})

				By("Waiting for the host pod to appear without injected nameservers", func() {
					Eventually(func(g Gomega) {
						hostPod, err := getHostPod(ctx, podName)
						g.Expect(err).To(Succeed(), "host pod for %s not yet present", podName)
						if hostPod.Spec.DNSConfig != nil {
							g.Expect(hostPod.Spec.DNSConfig.Nameservers).To(BeEmpty(),
								"hostNetwork pod should not have injected nameservers, got %v",
								hostPod.Spec.DNSConfig.Nameservers)
						}
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})

			It("respects user-defined dnsConfig.nameservers", func(ctx context.Context) {
				podName := "dns-pod-userdns-" + random.String(6)
				userIP := "8.8.8.8"

				By("Creating a pod with user-supplied dnsConfig in the vCluster", func() {
					createVPod(ctx, podName, func(p *corev1.Pod) {
						p.Spec.DNSPolicy = corev1.DNSNone
						p.Spec.DNSConfig = &corev1.PodDNSConfig{
							Nameservers: []string{userIP},
						}
					})
				})

				By("Waiting for the host pod and verifying the user nameservers are preserved", func() {
					Eventually(func(g Gomega) {
						hostPod, err := getHostPod(ctx, podName)
						g.Expect(err).To(Succeed(), "host pod for %s not yet present", podName)
						g.Expect(hostPod.Spec.DNSConfig).NotTo(BeNil(), "host pod dnsConfig is nil")
						g.Expect(hostPod.Spec.DNSConfig.Nameservers).To(Equal([]string{userIP}),
							"user-supplied nameservers were modified, got %v", hostPod.Spec.DNSConfig.Nameservers)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})

			It("resolves partial list when one Service is missing", func(ctx context.Context) {
				podName := "dns-pod-partial-" + random.String(6)

				By("Removing the dns-ns label from the secondary Service", func() {
					setServiceLabel(ctx, PodDNSNameserversSecondaryService, "")
					DeferCleanup(func(ctx context.Context) {
						setServiceLabel(ctx, PodDNSNameserversSecondaryService, "secondary")
					})
				})

				By("Creating a pod in the vCluster", func() {
					createVPod(ctx, podName, nil)
				})

				By("Waiting for the host pod to receive only the primary nameserver", func() {
					Eventually(func(g Gomega) {
						hostPod, err := getHostPod(ctx, podName)
						g.Expect(err).To(Succeed(), "host pod for %s not yet present", podName)
						g.Expect(hostPod.Spec.DNSConfig).NotTo(BeNil(), "host pod dnsConfig is nil")
						g.Expect(hostPod.Spec.DNSConfig.Nameservers).To(Equal([]string{primaryIP}),
							"expected only primary nameserver, got %v", hostPod.Spec.DNSConfig.Nameservers)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Waiting for a warning event on the virtual pod about the unresolved nameserver", func() {
					Eventually(func(g Gomega) {
						events, err := vClusterClient.CoreV1().Events("default").List(ctx, metav1.ListOptions{
							FieldSelector: "involvedObject.name=" + podName,
						})
						g.Expect(err).To(Succeed(), "failed to list events for pod %s", podName)
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
							podName, len(events.Items))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})

			It("blocks pod creation when all Services are missing", func(ctx context.Context) {
				podName := "dns-pod-blocked-" + random.String(6)

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

				By("Creating a pod in the vCluster", func() {
					createVPod(ctx, podName, nil)
				})

				By("Verifying the host pod is not created", func() {
					Consistently(func(g Gomega) {
						_, err := getHostPod(ctx, podName)
						g.Expect(kerrors.IsNotFound(err)).To(BeTrue(),
							"host pod for %s should not exist, got err=%v", podName, err)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
				})
			})

			// The embedded CoreDNS variant exercises the same nameservers
			// override but with controlPlane.coredns.embedded=true; it
			// requires its own vcluster-pods-dns-nameservers-embedded.yaml
			// and a dedicated suite_pods_dns_nameservers_embedded_test.go
			// so the lifecycle stays parallelisable. Tracked separately.
			PIt("works with embedded CoreDNS variant", Label("todo"), func(ctx context.Context) {})
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
