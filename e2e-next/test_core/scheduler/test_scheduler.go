package scheduler

import (
	"context"

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

// DescribeSchedulerTaintsAndTolerations registers scheduler taint/toleration tests.
// The vCluster must have virtualScheduler enabled and fromHost node sync with all: true.
func DescribeSchedulerTaintsAndTolerations(vcluster suite.Dependency) bool {
	return Describe("Scheduler sync - taints and tolerations",
		labels.Core,
		labels.Scheduler,
		cluster.Use(vcluster),
		cluster.Use(clusters.HostCluster),
		func() {
			var (
				hostClient     kubernetes.Interface
				vClusterClient kubernetes.Interface
			)

			BeforeEach(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
			})

			It("should use taints and tolerations to control pod scheduling on a virtual node", func(ctx context.Context) {
				suffix := random.String(6)
				taintKey := "e2e-taint-" + suffix

				taint := corev1.Taint{
					Key:    taintKey,
					Value:  "value1",
					Effect: corev1.TaintEffectNoSchedule,
				}

				// Pick the first virtual node to taint (only one, not all)
				virtualNodes, err := vClusterClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
				Expect(err).To(Succeed())
				Expect(virtualNodes.Items).NotTo(BeEmpty())
				targetNodeName := virtualNodes.Items[0].Name

				By("Adding a taint to one virtual node (not synced to host)", func() {
					err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
						node, err := vClusterClient.CoreV1().Nodes().Get(ctx, targetNodeName, metav1.GetOptions{})
						if err != nil {
							return err
						}
						node.Spec.Taints = append(node.Spec.Taints, taint)
						_, err = vClusterClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
						return err
					})
					Expect(err).To(Succeed())
				})
				DeferCleanup(func(ctx context.Context) {
					// Always remove the taint so other tests aren't affected
					err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
						node, err := vClusterClient.CoreV1().Nodes().Get(ctx, targetNodeName, metav1.GetOptions{})
						if err != nil {
							return err
						}
						filtered := make([]corev1.Taint, 0, len(node.Spec.Taints))
						for _, t := range node.Spec.Taints {
							if t.Key != taintKey {
								filtered = append(filtered, t)
							}
						}
						node.Spec.Taints = filtered
						_, err = vClusterClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
						return err
					})
					Expect(err).To(Succeed(), "failed to remove taint from node")
				})

				By("Verifying the taint is NOT on the host node", func() {
					hostNode, err := hostClient.CoreV1().Nodes().Get(ctx, targetNodeName, metav1.GetOptions{})
					Expect(err).To(Succeed())
					for _, t := range hostNode.Spec.Taints {
						Expect(t.Key).NotTo(Equal(taintKey),
							"taint %s should not be synced to host node %s", taintKey, targetNodeName)
					}
				})

				podWithToleration := "sched-tolerated-" + suffix
				podWithoutToleration := "sched-untolerated-" + suffix

				By("Creating a pod WITH matching toleration targeting the tainted node", func() {
					_, err := vClusterClient.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: podWithToleration},
						Spec: corev1.PodSpec{
							NodeSelector: map[string]string{"kubernetes.io/hostname": targetNodeName},
							Containers:   []corev1.Container{{Name: "nginx", Image: "nginx"}},
							Tolerations: []corev1.Toleration{{
								Key: taintKey, Operator: corev1.TolerationOpEqual,
								Value: "value1", Effect: corev1.TaintEffectNoSchedule,
							}},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Pods("default").Delete(ctx, podWithToleration, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				Eventually(func(g Gomega) {
					pod, err := vClusterClient.CoreV1().Pods("default").Get(ctx, podWithToleration, metav1.GetOptions{})
					g.Expect(err).To(Succeed())
					g.Expect(pod.Status.Phase).To(Equal(corev1.PodRunning),
						"pod with toleration should be Running, got %s", pod.Status.Phase)
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())

				By("Creating a pod WITHOUT toleration targeting the tainted node", func() {
					_, err := vClusterClient.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{Name: podWithoutToleration},
						Spec: corev1.PodSpec{
							NodeSelector: map[string]string{"kubernetes.io/hostname": targetNodeName},
							Containers:   []corev1.Container{{Name: "nginx", Image: "nginx"}},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Pods("default").Delete(ctx, podWithoutToleration, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})

				By("Verifying pod without toleration stays Pending", func() {
					Consistently(func(g Gomega) {
						pod, err := vClusterClient.CoreV1().Pods("default").Get(ctx, podWithoutToleration, metav1.GetOptions{})
						g.Expect(err).To(Succeed())
						g.Expect(pod.Status.Phase).NotTo(Equal(corev1.PodRunning),
							"pod without toleration should NOT be Running")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
				})
			})
		},
	)
}
