package test_scheduler

import (
	"context"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/platform/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Virtual scheduler taint/toleration scheduling",
	labels.Scheduler,
	cluster.Use(clusters.SchedulerVCluster),
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

		It("schedules a pod with matching toleration and blocks a pod without", func(ctx context.Context) {
			suffix := random.String(6)
			toleratedPodName := "scheduler-tolerated-" + suffix
			untolerantPodName := "scheduler-untolerant-" + suffix
			nsName := "default"

			taint := corev1.Taint{
				Key:    "e2e-scheduler-test",
				Value:  "block",
				Effect: corev1.TaintEffectNoSchedule,
			}

			By("Adding a custom taint to all virtual nodes", func() {
				virtualNodes, err := vClusterClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(virtualNodes.Items).NotTo(BeEmpty(), "Expected at least one virtual node")

				for _, vnode := range virtualNodes.Items {
					origNode := vnode.DeepCopy()
					vnode.Spec.Taints = append(vnode.Spec.Taints, taint)

					patch := client.MergeFrom(origNode)
					patchBytes, err := patch.Data(&vnode)
					Expect(err).NotTo(HaveOccurred())

					_, err = vClusterClient.CoreV1().Nodes().Patch(ctx, vnode.Name, patch.Type(), patchBytes, metav1.PatchOptions{})
					Expect(err).NotTo(HaveOccurred())
				}
			})

			// Register taint cleanup immediately so taints are removed even on test failure
			DeferCleanup(func(ctx context.Context) {
				virtualNodes, err := vClusterClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
				if err != nil {
					return
				}
				for _, vnode := range virtualNodes.Items {
					cleaned := make([]corev1.Taint, 0, len(vnode.Spec.Taints))
					for _, t := range vnode.Spec.Taints {
						if t.Key != taint.Key {
							cleaned = append(cleaned, t)
						}
					}
					if len(cleaned) == len(vnode.Spec.Taints) {
						continue
					}
					vnode.Spec.Taints = cleaned
					_, err = vClusterClient.CoreV1().Nodes().Update(ctx, &vnode, metav1.UpdateOptions{})
					Expect(err).NotTo(HaveOccurred())
				}
			})

			By("Verifying virtual node taints differ from host node taints", func() {
				hostNodes, err := hostClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())

				hostNodesTaints := make(map[string][]corev1.Taint)
				for _, hnode := range hostNodes.Items {
					hostNodesTaints[hnode.Name] = hnode.Spec.Taints
				}

				virtualNodes, err := vClusterClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())

				virtualNodesTaints := make(map[string][]corev1.Taint)
				for _, vnode := range virtualNodes.Items {
					virtualNodesTaints[vnode.Name] = vnode.Spec.Taints
				}

				Expect(virtualNodesTaints).NotTo(Equal(hostNodesTaints),
					"Virtual node taints should differ from host after adding custom taint")
			})

			By("Creating a pod with matching toleration", func() {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: toleratedPodName,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "nginx",
								Image: "nginx",
							},
						},
						Tolerations: []corev1.Toleration{
							{
								Key:      taint.Key,
								Operator: corev1.TolerationOpEqual,
								Value:    taint.Value,
								Effect:   taint.Effect,
							},
						},
					},
				}

				_, err := vClusterClient.CoreV1().Pods(nsName).Create(ctx, pod, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Pods(nsName).Delete(ctx, toleratedPodName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).NotTo(HaveOccurred())
					}
				})
			})

			By("Waiting for the tolerated pod to reach Running phase", func() {
				Eventually(func(g Gomega) {
					p, err := vClusterClient.CoreV1().Pods(nsName).Get(ctx, toleratedPodName, metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred(), "Failed to get tolerated pod")
					g.Expect(p.Status.Phase).To(Equal(corev1.PodRunning),
						fmt.Sprintf("Expected pod to be Running, got %s (reason: %s)", p.Status.Phase, p.Status.Reason))
				}).WithPolling(constants.PollingInterval).
					WithTimeout(constants.PollingTimeoutLong).
					Should(Succeed())
			})

			By("Creating a pod without the matching toleration", func() {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: untolerantPodName,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "nginx",
								Image: "nginx",
							},
						},
					},
				}

				_, err := vClusterClient.CoreV1().Pods(nsName).Create(ctx, pod, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Pods(nsName).Delete(ctx, untolerantPodName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).NotTo(HaveOccurred())
					}
				})
			})

			By("Verifying the non-tolerating pod stays Pending (unschedulable)", func() {
				Consistently(func(g Gomega) {
					p, err := vClusterClient.CoreV1().Pods(nsName).Get(ctx, untolerantPodName, metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred(), "Failed to get non-tolerating pod")
					g.Expect(p.Status.Phase).NotTo(Equal(corev1.PodRunning),
						"Non-tolerating pod should not reach Running phase")
				}).WithPolling(constants.PollingInterval).
					WithTimeout(constants.PollingTimeoutShort).
					Should(Succeed())
			})

			By("Removing the custom taint from all virtual nodes", func() {
				virtualNodes, err := vClusterClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())

				for _, vnode := range virtualNodes.Items {
					cleaned := make([]corev1.Taint, 0, len(vnode.Spec.Taints))
					for _, t := range vnode.Spec.Taints {
						if t.Key != taint.Key {
							cleaned = append(cleaned, t)
						}
					}
					vnode.Spec.Taints = cleaned
					_, err = vClusterClient.CoreV1().Nodes().Update(ctx, &vnode, metav1.UpdateOptions{})
					Expect(err).NotTo(HaveOccurred())
				}
			})

			By("Verifying virtual node taints match host node taints after cleanup", func() {
				hostNodes, err := hostClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())

				hostNodesTaints := make(map[string][]corev1.Taint)
				for _, hnode := range hostNodes.Items {
					hostNodesTaints[hnode.Name] = hnode.Spec.Taints
				}

				Eventually(func(g Gomega) {
					virtualNodes, err := vClusterClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
					g.Expect(err).NotTo(HaveOccurred())

					virtualNodesTaints := make(map[string][]corev1.Taint)
					for _, vnode := range virtualNodes.Items {
						virtualNodesTaints[vnode.Name] = vnode.Spec.Taints
					}

					g.Expect(virtualNodesTaints).To(Equal(hostNodesTaints),
						"Virtual node taints should match host after removing custom taint")
				}).WithPolling(constants.PollingInterval).
					WithTimeout(constants.PollingTimeout).
					Should(Succeed())
			})
		})
	},
)
