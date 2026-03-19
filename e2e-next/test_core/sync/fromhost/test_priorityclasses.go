package fromhost

import (
	"context"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var _ = Describe("PriorityClasses sync from host",
	labels.Core,
	labels.Sync,
	labels.PriorityClasses,
	cluster.Use(clusters.FromHostLimitClassesVCluster),
	func() {
		var (
			hostClient     kubernetes.Interface
			vClusterClient kubernetes.Interface
			vClusterName   = clusters.FromHostLimitClassesVClusterName
			vClusterHostNS = "vcluster-" + clusters.FromHostLimitClassesVClusterName
		)

		BeforeEach(func(ctx context.Context) {
			hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
			Expect(hostClient).NotTo(BeNil())
			vClusterClient = cluster.CurrentKubeClientFrom(ctx)
			Expect(vClusterClient).NotTo(BeNil())
		})

		// createPriorityClass creates a PriorityClass on the host and registers cleanup.
		// Returns the created object for further assertions.
		createPriorityClass := func(ctx context.Context, name string, value int32, pcLabels map[string]string) *schedulingv1.PriorityClass {
			GinkgoHelper()
			pc := &schedulingv1.PriorityClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:   name,
					Labels: pcLabels,
				},
				Value:         value,
				GlobalDefault: false,
			}
			created, err := hostClient.SchedulingV1().PriorityClasses().Create(ctx, pc, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				err := hostClient.SchedulingV1().PriorityClasses().Delete(ctx, name, metav1.DeleteOptions{})
				if !kerrors.IsNotFound(err) {
					Expect(err).To(Succeed())
				}
			})
			return created
		}

		It("only syncs priorityClasses matching the label selector to vcluster", func(ctx context.Context) {
			suffix := fmt.Sprintf("%d", GinkgoRandomSeed())
			matchingName := "pc-match-" + suffix
			nonMatchingName := "pc-nomatch-" + suffix

			createPriorityClass(ctx, matchingName, 1000000, map[string]string{"value": "one"})
			createPriorityClass(ctx, nonMatchingName, 10000, map[string]string{"value": "two"})

			By("waiting for the matching class to appear and the non-matching class to stay absent", func() {
				Eventually(func(g Gomega) {
					priorityClasses, err := vClusterClient.SchedulingV1().PriorityClasses().List(ctx, metav1.ListOptions{})
					g.Expect(err).To(Succeed(), "failed to list priorityClasses in vcluster: %v", err)

					var foundMatch, foundNoMatch bool
					for _, pc := range priorityClasses.Items {
						switch pc.Name {
						case matchingName:
							foundMatch = true
						case nonMatchingName:
							foundNoMatch = true
						}
					}
					g.Expect(foundMatch).To(BeTrue(), "expected matching priorityClass to be synced to vcluster")
					g.Expect(foundNoMatch).To(BeFalse(), "expected non-matching priorityClass to stay absent from vcluster")
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})
		})

		It("rejects pod creation in vcluster using a priorityClass not synced from host", func(ctx context.Context) {
			suffix := fmt.Sprintf("%d", GinkgoRandomSeed())
			nonMatchingName := "pc-reject-" + suffix

			createPriorityClass(ctx, nonMatchingName, 10000, map[string]string{"value": "two"})

			By("attempting to create a pod using the non-synced priorityClass in vcluster", func() {
				_, err := vClusterClient.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "reject-pod-" + suffix,
						Namespace: "default",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "nginx", Image: "nginx"},
						},
						PriorityClassName: nonMatchingName,
					},
				}, metav1.CreateOptions{})
				Expect(err).To(MatchError(ContainSubstring(
					fmt.Sprintf("no PriorityClass with name %s was found", nonMatchingName),
				)))
			})
		})

		It("syncs pods created in vcluster to host when using a priorityClass synced from host", func(ctx context.Context) {
			suffix := fmt.Sprintf("%d", GinkgoRandomSeed())
			matchingName := "pc-podsync-" + suffix
			podName := "hp-pod-" + suffix

			createPriorityClass(ctx, matchingName, 1000000, map[string]string{"value": "one"})

			By("waiting for the priorityClass to be synced to vcluster", func() {
				Eventually(func(g Gomega) {
					_, err := vClusterClient.SchedulingV1().PriorityClasses().Get(ctx, matchingName, metav1.GetOptions{})
					g.Expect(err).To(Succeed(), "priorityClass %s not yet synced to vcluster: %v", matchingName, err)
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})

			By("creating a pod using the synced priorityClass in vcluster", func() {
				_, err := vClusterClient.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      podName,
						Namespace: "default",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "nginx", Image: "nginx"},
						},
						PriorityClassName: matchingName,
					},
				}, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Pods("default").Delete(ctx, podName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})
			})

			By("waiting for the pod to appear in the host vcluster namespace", func() {
				expectedHostPodName := translate.SafeConcatName(podName, "x", "default", "x", vClusterName)
				Eventually(func(g Gomega) {
					pods, err := hostClient.CoreV1().Pods(vClusterHostNS).List(ctx, metav1.ListOptions{})
					g.Expect(err).To(Succeed(), "failed to list pods in host namespace %s: %v", vClusterHostNS, err)
					var found bool
					for _, pod := range pods.Items {
						if pod.Name == expectedHostPodName {
							found = true
							break
						}
					}
					g.Expect(found).To(BeTrue(), "expected pod %s to appear in host namespace %s", expectedHostPodName, vClusterHostNS)
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})
		})

		It("propagates description updates from host priorityClass to vcluster", func(ctx context.Context) {
			suffix := fmt.Sprintf("%d", GinkgoRandomSeed())
			pcName := "pc-update-" + suffix
			updatedDescription := "Updated description."

			createPriorityClass(ctx, pcName, 1000000, map[string]string{"value": "one"})

			By("waiting for the priorityClass to be synced to vcluster", func() {
				Eventually(func(g Gomega) {
					_, err := vClusterClient.SchedulingV1().PriorityClasses().Get(ctx, pcName, metav1.GetOptions{})
					g.Expect(err).To(Succeed(), "priorityClass %s not yet synced to vcluster: %v", pcName, err)
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})

			By("updating the priorityClass description on host", func() {
				pc, err := hostClient.SchedulingV1().PriorityClasses().Get(ctx, pcName, metav1.GetOptions{})
				Expect(err).To(Succeed())
				pc.Description = updatedDescription
				_, err = hostClient.SchedulingV1().PriorityClasses().Update(ctx, pc, metav1.UpdateOptions{})
				Expect(err).To(Succeed())
			})

			By("waiting for the updated description to appear in vcluster", func() {
				Eventually(func(g Gomega) {
					pc, err := vClusterClient.SchedulingV1().PriorityClasses().Get(ctx, pcName, metav1.GetOptions{})
					g.Expect(err).To(Succeed(), "failed to get priorityClass %s from vcluster: %v", pcName, err)
					g.Expect(pc.Description).To(Equal(updatedDescription),
						"expected vcluster priorityClass description to match updated host value")
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})
		})

		It("removes synced priorityClass from vcluster when host label no longer matches selector", func(ctx context.Context) {
			suffix := fmt.Sprintf("%d", GinkgoRandomSeed())
			pcName := "pc-labeldel-" + suffix

			createPriorityClass(ctx, pcName, 1000000, map[string]string{"value": "one"})

			By("waiting for the priorityClass to be synced to vcluster", func() {
				Eventually(func(g Gomega) {
					_, err := vClusterClient.SchedulingV1().PriorityClasses().Get(ctx, pcName, metav1.GetOptions{})
					g.Expect(err).To(Succeed(), "priorityClass %s not yet synced to vcluster: %v", pcName, err)
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})

			By("removing the matching label from the priorityClass on host", func() {
				pc, err := hostClient.SchedulingV1().PriorityClasses().Get(ctx, pcName, metav1.GetOptions{})
				Expect(err).To(Succeed())
				delete(pc.Labels, "value")
				_, err = hostClient.SchedulingV1().PriorityClasses().Update(ctx, pc, metav1.UpdateOptions{})
				Expect(err).To(Succeed())
			})

			By("waiting for the priorityClass to disappear from vcluster", func() {
				Eventually(func(g Gomega) {
					_, err := vClusterClient.SchedulingV1().PriorityClasses().Get(ctx, pcName, metav1.GetOptions{})
					g.Expect(kerrors.IsNotFound(err)).To(BeTrue(),
						"expected priorityClass to be removed from vcluster after label mismatch, got: %v", err)
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})
		})
	},
)
