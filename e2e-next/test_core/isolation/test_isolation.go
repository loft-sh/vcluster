package isolation

import (
	"context"
	"fmt"
	"strings"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DescribeIsolationMode registers isolation mode tests against the given vCluster.
// The vCluster must be configured with policies.podSecurityStandard, resourceQuota,
// limitRange, and networkPolicy enabled (see vcluster-isolation-mode.yaml).
func DescribeIsolationMode(vcluster suite.Dependency) bool {
	return Describe("Isolated mode",
		labels.Core,
		labels.Security,
		cluster.Use(vcluster),
		cluster.Use(clusters.HostCluster),
		func() {
			var (
				hostClient        kubernetes.Interface
				vClusterClient    kubernetes.Interface
				vClusterName      string
				vClusterNamespace string
			)

			BeforeEach(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				// Host namespace follows the pattern "vcluster-<name>" (see vcluster-isolation-mode.yaml).
				vClusterNamespace = "vcluster-" + vClusterName
			})

			It("enforces isolated mode", func(ctx context.Context) {
				By("Checking if isolated mode creates a ResourceQuota on the host", func() {
					resourceQuotaName := "vc-" + vClusterName
					_, err := hostClient.CoreV1().ResourceQuotas(vClusterNamespace).Get(ctx, resourceQuotaName, metav1.GetOptions{})
					Expect(err).To(Succeed(), "ResourceQuota %s not found in host namespace %s", resourceQuotaName, vClusterNamespace)
				})

				By("Checking if isolated mode creates a LimitRange on the host", func() {
					limitRangeName := "vc-" + vClusterName
					_, err := hostClient.CoreV1().LimitRanges(vClusterNamespace).Get(ctx, limitRangeName, metav1.GetOptions{})
					Expect(err).To(Succeed(), "LimitRange %s not found in host namespace %s", limitRangeName, vClusterNamespace)
				})

				By("Checking if isolated mode creates a NetworkPolicy on the host", func() {
					networkPolicyName := "vc-work-" + vClusterName
					_, err := hostClient.NetworkingV1().NetworkPolicies(vClusterNamespace).Get(ctx, networkPolicyName, metav1.GetOptions{})
					Expect(err).To(Succeed(), "NetworkPolicy %s not found in host namespace %s", networkPolicyName, vClusterNamespace)
				})

				By("Checking if isolated mode applies baseline PodSecurityStandards to namespaces in vcluster", func() {
					ns, err := vClusterClient.CoreV1().Namespaces().Get(ctx, "default", metav1.GetOptions{})
					Expect(err).To(Succeed())
					Expect(ns.Labels).To(HaveKeyWithValue("pod-security.kubernetes.io/enforce", "baseline"),
						"baseline PodSecurityStandards not applied to default namespace, labels: %v", ns.Labels)
				})

				By("Checking if isolated mode applies baseline PodSecurityStandards to new namespace in vcluster", func() {
					suffix := random.String(6)
					nsName := "isolation-pss-" + suffix
					_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: nsName},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := vClusterClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).To(Succeed())
						}
					})

					Eventually(func(g Gomega) {
						ns, err := vClusterClient.CoreV1().Namespaces().Get(ctx, nsName, metav1.GetOptions{})
						g.Expect(err).To(Succeed(), "failed to get namespace %s: %v", nsName, err)
						g.Expect(ns.Status.Phase).To(Equal(corev1.NamespaceActive),
							"namespace %s phase is %s, waiting for Active", nsName, ns.Status.Phase)
						g.Expect(ns.Labels).To(HaveKeyWithValue("pod-security.kubernetes.io/enforce", "baseline"),
							"baseline PodSecurityStandards not applied to namespace %s, labels: %v", nsName, ns.Labels)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("fails to schedule pod violating resourcequota and limitrange", func(ctx context.Context) {
				suffix := random.String(6)
				podName := "quota-violator-" + suffix

				_, err := vClusterClient.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pod",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: podName,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "nginx",
								Image: "nginx",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU: resource.MustParse("2"),
									},
								},
							},
						},
					},
				}, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Pods("default").Delete(ctx, podName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				By("Waiting for a LimitRange violation event for the over-quota pod", func() {
					Eventually(func(g Gomega) {
						p, err := vClusterClient.CoreV1().Pods("default").Get(ctx, podName, metav1.GetOptions{})
						g.Expect(err).To(Succeed(), "failed to get pod %s: %v", podName, err)

						if p.Status.Phase == corev1.PodRunning {
							// If the pod reached Running, the LimitRange is not being enforced.
							Fail(fmt.Sprintf("pod %s reached Running state unexpectedly - LimitRange should block it", podName))
						}

						events, err := vClusterClient.CoreV1().Events("default").List(ctx, metav1.ListOptions{
							FieldSelector: "involvedObject.name=" + podName + ",involvedObject.kind=Pod",
						})
						g.Expect(err).To(Succeed(), "failed to list events for pod %s: %v", podName, err)

						var found bool
						for _, e := range events.Items {
							if strings.Contains(e.Message, `Invalid value: "2": must be less than or equal to cpu limit`) {
								found = true
								break
							}
						}
						g.Expect(found).To(BeTrue(),
							"expected LimitRange violation event for pod %s (cpu request 2 exceeds limit), events: %v", podName, events.Items)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})
		},
	)
}
