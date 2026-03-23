package scheduler

import (
	"context"
	"reflect"

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
)

// DescribeSchedulerTaintsAndTolerations registers scheduler taint/toleration tests against the given vCluster.
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

			// Ordered because specs form a lifecycle sequence:
			// spec 1 adds taints and verifies a tolerating pod runs,
			// spec 2 verifies a non-tolerating pod does NOT run (taints still present),
			// spec 3 removes the taints and verifies taint state is restored.
			// Each spec depends on taint state written by the previous spec.
			Context("taint lifecycle", Ordered, func() {
				var (
					hostNodesTaints map[string][]corev1.Taint
				)

				It("adds taints to virtual nodes only and verifies host nodes are unaffected", func(ctx context.Context) {
					By("Adding taints to virtual nodes", func() {
						virtualNodes, err := vClusterClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
						Expect(err).To(Succeed())
						Expect(virtualNodes.Items).NotTo(BeEmpty(), "expected at least one virtual node")

						for _, vnode := range virtualNodes.Items {
							updated := vnode.DeepCopy()
							updated.Spec.Taints = append(updated.Spec.Taints, corev1.Taint{
								Key:    "key1",
								Value:  "value1",
								Effect: corev1.TaintEffectNoSchedule,
							})
							_, err = vClusterClient.CoreV1().Nodes().Update(ctx, updated, metav1.UpdateOptions{})
							Expect(err).To(Succeed(), "failed to update taints on virtual node %s", vnode.Name)
						}
					})

					By("Capturing host node taint state for later comparison", func() {
						hostNodes, err := hostClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
						Expect(err).To(Succeed())

						hostNodesTaints = make(map[string][]corev1.Taint)
						for _, hnode := range hostNodes.Items {
							hostNodesTaints[hnode.Name] = hnode.Spec.Taints
						}
					})

					By("Verifying virtual node taints differ from host node taints", func() {
						virtualNodes, err := vClusterClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
						Expect(err).To(Succeed())

						virtualNodesTaints := make(map[string][]corev1.Taint)
						for _, vnode := range virtualNodes.Items {
							virtualNodesTaints[vnode.Name] = vnode.Spec.Taints
						}

						Expect(reflect.DeepEqual(hostNodesTaints, virtualNodesTaints)).To(BeFalse(),
							"host and virtual node taints should differ after adding taints to virtual nodes only")
					})
				})

				It("schedules a pod with a matching toleration", func(ctx context.Context) {
					suffix := random.String(6)
					podName := "nginx-toleration-" + suffix

					_, err := vClusterClient.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
						TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
						ObjectMeta: metav1.ObjectMeta{
							Name: podName,
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "nginx", Image: "nginx"},
							},
							Tolerations: []corev1.Toleration{
								{
									Key:      "key1",
									Operator: corev1.TolerationOpEqual,
									Value:    "value1",
									Effect:   corev1.TaintEffectNoSchedule,
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

					By("Waiting for pod with matching toleration to reach Running phase", func() {
						Eventually(func(g Gomega) {
							p, err := vClusterClient.CoreV1().Pods("default").Get(ctx, podName, metav1.GetOptions{})
							g.Expect(err).To(Succeed(), "failed to get pod %s", podName)
							g.Expect(p.Status.Phase).To(Equal(corev1.PodRunning),
								"pod %s is in phase %s, waiting for Running", podName, p.Status.Phase)
						}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
					})
				})

				It("does not schedule a pod without a matching toleration", func(ctx context.Context) {
					suffix := random.String(6)
					podName := "nginx-notoleration-" + suffix

					_, err := vClusterClient.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
						TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
						ObjectMeta: metav1.ObjectMeta{
							Name: podName,
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "nginx", Image: "nginx"},
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

					By("Verifying pod without toleration remains unscheduled (not Running)", func() {
						Consistently(func(g Gomega) {
							p, err := vClusterClient.CoreV1().Pods("default").Get(ctx, podName, metav1.GetOptions{})
							g.Expect(err).To(Succeed(), "failed to get pod %s", podName)
							g.Expect(p.Status.Phase).NotTo(Equal(corev1.PodRunning),
								"pod %s unexpectedly reached Running phase despite missing toleration", podName)
						}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
					})
				})

				It("removes taints from virtual nodes and restores parity with host nodes", func(ctx context.Context) {
					By("Removing the added taint from each virtual node", func() {
						vNodes, err := vClusterClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
						Expect(err).To(Succeed())

						for _, vnode := range vNodes.Items {
							updated := vnode.DeepCopy()
							// Remove the last taint (the one added in the first spec)
							if len(updated.Spec.Taints) > 0 {
								updated.Spec.Taints = updated.Spec.Taints[:len(updated.Spec.Taints)-1]
							}
							_, err = vClusterClient.CoreV1().Nodes().Update(ctx, updated, metav1.UpdateOptions{})
							Expect(err).To(Succeed(), "failed to remove taint from virtual node %s", vnode.Name)
						}
					})

					By("Verifying virtual node taints match host node taints again", func() {
						Eventually(func(g Gomega) {
							virtualNodes, err := vClusterClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
							g.Expect(err).To(Succeed(), "failed to list virtual nodes")

							virtualNodesTaints := make(map[string][]corev1.Taint)
							for _, vnode := range virtualNodes.Items {
								virtualNodesTaints[vnode.Name] = vnode.Spec.Taints
							}

							g.Expect(reflect.DeepEqual(hostNodesTaints, virtualNodesTaints)).To(BeTrue(),
								"virtual node taints should match host node taints after removal; host=%v virtual=%v",
								hostNodesTaints, virtualNodesTaints)
						}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
					})
				})
			})
		},
	)
}
