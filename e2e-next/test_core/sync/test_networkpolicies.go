package test_core

import (
	"context"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	pkgconstants "github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/util/podhelper"
	"github.com/loft-sh/vcluster/pkg/util/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
)

// NetworkPolicyEnforcementSpec verifies that NetworkPolicy egress rules are correctly
// enforced in the virtual cluster. It requires a CNI with NetworkPolicy support (e.g. Calico).
// Standard Kind uses kindnet which does NOT enforce NetworkPolicies.
// This test runs in a dedicated CI job (suite_networkpolicies_test.go) on a Kind
// cluster with Calico CNI installed.
func NetworkPolicyEnforcementSpec() {
	Describe("NetworkPolicy egress enforcement",
		labels.NetworkPolicies,
		func() {
			var (
				vClusterClient kubernetes.Interface
				vClusterConfig *rest.Config
			)

			BeforeEach(func(ctx context.Context) {
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				currentClusterName := cluster.CurrentClusterNameFrom(ctx)
				vClusterConfig = cluster.From(ctx, currentClusterName).KubernetesRestConfig()
				Expect(vClusterConfig).NotTo(BeNil())
			})

			It("enforces egress NetworkPolicy rules between virtual namespaces", func(ctx context.Context) {
				suffix := random.String(6)
				nsNameA := "netpol-egress-a-" + suffix
				nsNameB := "netpol-egress-b-" + suffix

				By("Creating virtual namespace A (curl pod namespace)", func() {
					_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name:   nsNameA,
							Labels: map[string]string{"key-a": "netpol-label-a-" + suffix},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})
				DeferCleanup(func(ctx context.Context) {
					propagationPolicy := metav1.DeletePropagationBackground
					err := vClusterClient.CoreV1().Namespaces().Delete(ctx, nsNameA, metav1.DeleteOptions{
						PropagationPolicy: &propagationPolicy,
					})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				By("Creating virtual namespace B (nginx pod namespace)", func() {
					_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name:   nsNameB,
							Labels: map[string]string{"key-b": "netpol-label-b-" + suffix},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})
				DeferCleanup(func(ctx context.Context) {
					propagationPolicy := metav1.DeletePropagationBackground
					err := vClusterClient.CoreV1().Namespaces().Delete(ctx, nsNameB, metav1.DeleteOptions{
						PropagationPolicy: &propagationPolicy,
					})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				nginxLabels := map[string]string{"app": "nginx-" + suffix}
				var nginxService *corev1.Service

				By("Creating the nginx pod in namespace B", func() {
					_, err := vClusterClient.CoreV1().Pods(nsNameB).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name:   "nginx-" + suffix,
							Labels: nginxLabels,
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            "nginx",
									Image:           "nginxinc/nginx-unprivileged:stable-alpine3.20-slim",
									ImagePullPolicy: corev1.PullIfNotPresent,
									SecurityContext: &corev1.SecurityContext{
										RunAsUser: ptr.To(int64(12345)),
									},
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				By("Creating the nginx service in namespace B", func() {
					var err error
					nginxService, err = vClusterClient.CoreV1().Services(nsNameB).Create(ctx, &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "nginx-" + suffix,
							Namespace: nsNameB,
						},
						Spec: corev1.ServiceSpec{
							Selector: nginxLabels,
							Ports: []corev1.ServicePort{
								{Port: 8080},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				curlPodName := "curl-" + suffix

				By("Creating the curl pod in namespace A", func() {
					_, err := vClusterClient.CoreV1().Pods(nsNameA).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: curlPodName},
						Spec: corev1.PodSpec{
							TerminationGracePeriodSeconds: ptr.To(int64(1)),
							Containers: []corev1.Container{
								{
									Name:            "curl",
									Image:           "curlimages/curl",
									ImagePullPolicy: corev1.PullIfNotPresent,
									SecurityContext: &corev1.SecurityContext{
										RunAsUser: ptr.To(int64(12345)),
									},
									Command: []string{"sleep"},
									Args:    []string{"9999"},
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				By("Waiting for nginx pod to reach Running phase", func() {
					Eventually(func(g Gomega) {
						pod, err := vClusterClient.CoreV1().Pods(nsNameB).Get(ctx, "nginx-"+suffix, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "failed to get nginx pod: %v", err)
						g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning),
							"nginx pod phase is %s, not yet Running", pod.Status.Phase)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				By("Waiting for curl pod to reach Running phase", func() {
					Eventually(func(g Gomega) {
						pod, err := vClusterClient.CoreV1().Pods(nsNameA).Get(ctx, curlPodName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), "failed to get curl pod: %v", err)
						g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning),
							"curl pod phase is %s, not yet Running", pod.Status.Phase)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})

				// curlService sends a single HTTP request from the curl pod to the nginx service
				// and returns (httpCode, stderr, error).
				curlService := func(ctx context.Context) (string, string, error) {
					GinkgoHelper()
					url := fmt.Sprintf("http://%s.%s.svc:%d/", nginxService.GetName(), nginxService.GetNamespace(), nginxService.Spec.Ports[0].Port)
					cmd := []string{"curl", "-s", "--show-error", "-o", "/dev/null", "-w", "%{http_code}", "--max-time", "2", url}
					stdout, stderr, err := podhelper.ExecBuffered(ctx, vClusterConfig, nsNameA, curlPodName, "curl", cmd, nil)
					return string(stdout), string(stderr), err
				}

				// testServiceReachable polls until the curl pod receives a 200 from the nginx service.
				testServiceReachable := func(ctx context.Context) {
					GinkgoHelper()
					Eventually(func(g Gomega) {
						httpCode, _, err := curlService(ctx)
						g.Expect(err).NotTo(HaveOccurred(), "curl exec failed")
						g.Expect(httpCode).To(Equal("200"), "expected 200 from nginx service, got %s", httpCode)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				}

				// testServiceUnreachable polls until the curl pod receives a connection timeout (000).
				testServiceUnreachable := func(ctx context.Context) {
					GinkgoHelper()
					Eventually(func(g Gomega) {
						httpCode, _, _ := curlService(ctx)
						g.Expect(httpCode).To(Equal("000"), "expected timeout (000) from nginx service, got %s", httpCode)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				}

				// updateNetworkPolicy fetches the current policy, applies the mutator, and updates it.
				// Retries on conflict.
				updateNetworkPolicy := func(ctx context.Context, np *networkingv1.NetworkPolicy, mutator func(*networkingv1.NetworkPolicy)) {
					GinkgoHelper()
					err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
						var err error
						np, err = vClusterClient.NetworkingV1().NetworkPolicies(np.GetNamespace()).Get(ctx, np.GetName(), metav1.GetOptions{})
						if err != nil {
							return err
						}
						mutator(np)
						np, err = vClusterClient.NetworkingV1().NetworkPolicies(np.GetNamespace()).Update(ctx, np, metav1.UpdateOptions{})
						return err
					})
					Expect(err).To(Succeed())
				}

				By("Verifying service is reachable before any NetworkPolicy is applied", func() {
					testServiceReachable(ctx)
				})

				By("Creating allow-coredns-egress NetworkPolicy in namespace A so DNS continues to work", func() {
					UDPProtocol := corev1.ProtocolUDP
					_, err := vClusterClient.NetworkingV1().NetworkPolicies(nsNameA).Create(ctx, &networkingv1.NetworkPolicy{
						ObjectMeta: metav1.ObjectMeta{Namespace: nsNameA, Name: "allow-coredns-egress"},
						Spec: networkingv1.NetworkPolicySpec{
							PodSelector: metav1.LabelSelector{},
							Egress: []networkingv1.NetworkPolicyEgressRule{
								{
									Ports: []networkingv1.NetworkPolicyPort{
										{
											Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: 1053},
											Protocol: &UDPProtocol,
										},
									},
									To: []networkingv1.NetworkPolicyPeer{
										{
											PodSelector: &metav1.LabelSelector{
												MatchLabels: map[string]string{
													pkgconstants.CoreDNSLabelKey: pkgconstants.CoreDNSLabelValue,
												},
											},
											NamespaceSelector: &metav1.LabelSelector{
												MatchLabels: map[string]string{
													"kubernetes.io/metadata.name": "kube-system",
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

				var egressPolicy *networkingv1.NetworkPolicy

				By("Creating a deny-all-egress NetworkPolicy in namespace A", func() {
					var err error
					egressPolicy, err = vClusterClient.NetworkingV1().NetworkPolicies(nsNameA).Create(ctx, &networkingv1.NetworkPolicy{
						ObjectMeta: metav1.ObjectMeta{Namespace: nsNameA, Name: "my-egress-policy"},
						Spec: networkingv1.NetworkPolicySpec{
							PodSelector: metav1.LabelSelector{},
							PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				By("Verifying service is unreachable after deny-all-egress policy is applied", func() {
					testServiceUnreachable(ctx)
				})

				nsB, err := vClusterClient.CoreV1().Namespaces().Get(ctx, nsNameB, metav1.GetOptions{})
				Expect(err).To(Succeed())

				nginxPod, err := vClusterClient.CoreV1().Pods(nsNameB).Get(ctx, "nginx-"+suffix, metav1.GetOptions{})
				Expect(err).To(Succeed())

				By("Updating egress policy to allow traffic to namespace B by namespace selector", func() {
					updateNetworkPolicy(ctx, egressPolicy, func(np *networkingv1.NetworkPolicy) {
						np.Spec.Egress = []networkingv1.NetworkPolicyEgressRule{
							{
								Ports: []networkingv1.NetworkPolicyPort{
									{Port: &intstr.IntOrString{Type: intstr.Int, IntVal: nginxService.Spec.Ports[0].Port}},
								},
								To: []networkingv1.NetworkPolicyPeer{
									{
										NamespaceSelector: &metav1.LabelSelector{
											MatchLabels: nsB.GetLabels(),
										},
									},
								},
							},
						}
					})
				})

				By("Verifying service is reachable when egress to namespace B is allowed", func() {
					testServiceReachable(ctx)
				})

				By("Updating egress policy to deny traffic using a non-matching pod selector", func() {
					updateNetworkPolicy(ctx, egressPolicy, func(np *networkingv1.NetworkPolicy) {
						np.Spec.Egress = []networkingv1.NetworkPolicyEgressRule{
							{
								Ports: []networkingv1.NetworkPolicyPort{
									{Port: &intstr.IntOrString{Type: intstr.Int, IntVal: nginxService.Spec.Ports[0].Port}},
								},
								To: []networkingv1.NetworkPolicyPeer{
									{
										NamespaceSelector: &metav1.LabelSelector{
											MatchLabels: nsB.GetLabels(),
										},
										PodSelector: &metav1.LabelSelector{
											MatchLabels: map[string]string{"wrong": "label"},
										},
									},
								},
							},
						}
					})
				})

				By("Verifying service is unreachable when pod selector does not match nginx pod", func() {
					testServiceUnreachable(ctx)
				})

				By("Updating egress policy to allow traffic to namespace B and nginx pod selector", func() {
					updateNetworkPolicy(ctx, egressPolicy, func(np *networkingv1.NetworkPolicy) {
						np.Spec.Egress = []networkingv1.NetworkPolicyEgressRule{
							{
								Ports: []networkingv1.NetworkPolicyPort{
									{Port: &intstr.IntOrString{Type: intstr.Int, IntVal: nginxService.Spec.Ports[0].Port}},
								},
								To: []networkingv1.NetworkPolicyPeer{
									{
										NamespaceSelector: &metav1.LabelSelector{
											MatchLabels: nsB.GetLabels(),
										},
										PodSelector: &metav1.LabelSelector{
											MatchLabels: nginxPod.GetLabels(),
										},
									},
								},
							},
						}
					})
				})

				By("Verifying service is reachable when both namespace and pod selectors match", func() {
					testServiceReachable(ctx)
				})

				By("Updating egress policy to deny traffic by targeting a wrong port", func() {
					updateNetworkPolicy(ctx, egressPolicy, func(np *networkingv1.NetworkPolicy) {
						np.Spec.Egress = []networkingv1.NetworkPolicyEgressRule{
							{
								Ports: []networkingv1.NetworkPolicyPort{
									// Allow only port 1; nginx listens on 8080, so traffic is denied.
									{Port: &intstr.IntOrString{Type: intstr.Int, IntVal: 1}},
								},
								To: []networkingv1.NetworkPolicyPeer{
									{
										NamespaceSelector: &metav1.LabelSelector{
											MatchLabels: nsB.GetLabels(),
										},
										PodSelector: &metav1.LabelSelector{
											MatchLabels: nginxPod.GetLabels(),
										},
									},
								},
							},
						}
					})
				})

				By("Verifying service is unreachable when allowed port does not match nginx port", func() {
					testServiceUnreachable(ctx)
				})
			})
		},
	)
}
