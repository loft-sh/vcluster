package fromhost

import (
	"context"
	"fmt"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	nodev1 "k8s.io/api/node/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DescribeFromHostRuntimeClasses registers runtimeClass sync from host tests against the given vCluster.
func DescribeFromHostRuntimeClasses(vcluster suite.Dependency) bool {
	return Describe("RuntimeClasses sync from host",
		labels.Core,
		labels.PR,
		labels.Sync,
		labels.RuntimeClasses,
		cluster.Use(vcluster),
		func() {
			var (
				hostClient     kubernetes.Interface
				vClusterClient kubernetes.Interface
				vClusterName   string
				vClusterHostNS string
			)

			BeforeEach(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterHostNS = "vcluster-" + vClusterName
			})

			// createRuntimeClass creates a RuntimeClass on the host and registers cleanup.
			// Returns the created object for further assertions.
			createRuntimeClass := func(ctx context.Context, name string, handler string, rcLabels map[string]string) *nodev1.RuntimeClass {
				GinkgoHelper()
				rc := &nodev1.RuntimeClass{
					ObjectMeta: metav1.ObjectMeta{
						Name:   name,
						Labels: rcLabels,
					},
					Handler: handler,
				}
				created, err := hostClient.NodeV1().RuntimeClasses().Create(ctx, rc, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				DeferCleanup(func(ctx context.Context) {
					err := hostClient.NodeV1().RuntimeClasses().Delete(ctx, name, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})
				return created
			}

			It("only syncs runtimeClasses matching the label selector to vcluster", func(ctx context.Context) {
				suffix := random.String(6)
				matchingName := "rc-match-" + suffix
				nonMatchingName := "rc-nomatch-" + suffix

				createRuntimeClass(ctx, matchingName, "runc", map[string]string{"value": "one"})
				createRuntimeClass(ctx, nonMatchingName, "runsc", map[string]string{"value": "two"})

				By("waiting for the matching class to appear and the non-matching class to stay absent", func() {
					Eventually(func(g Gomega) {
						runtimeClasses, err := vClusterClient.NodeV1().RuntimeClasses().List(ctx, metav1.ListOptions{})
						g.Expect(err).To(Succeed(), "failed to list runtimeClasses in vcluster: %v", err)

						var foundMatch, foundNoMatch bool
						for _, rc := range runtimeClasses.Items {
							switch rc.Name {
							case matchingName:
								foundMatch = true
							case nonMatchingName:
								foundNoMatch = true
							}
						}
						g.Expect(foundMatch).To(BeTrue(), "expected matching runtimeClass to be synced to vcluster")
						g.Expect(foundNoMatch).To(BeFalse(), "expected non-matching runtimeClass to stay absent from vcluster")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			It("rejects pod creation in vcluster using a runtimeClass not synced from host", func(ctx context.Context) {
				suffix := random.String(6)
				nonMatchingName := "rc-reject-" + suffix

				createRuntimeClass(ctx, nonMatchingName, "runsc", map[string]string{"value": "two"})

				By("attempting to create a pod using the non-synced runtimeClass in vcluster", func() {
					podName := "reject-pod-" + suffix
					_, err := vClusterClient.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name:      podName,
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "nginx", Image: "nginx"},
							},
							RuntimeClassName: &nonMatchingName,
						},
					}, metav1.CreateOptions{})
					expectedMsg := fmt.Sprintf(`pods "%s" is forbidden: pod rejected: RuntimeClass "%s" not found`, podName, nonMatchingName)
					Expect(err).To(MatchError(ContainSubstring(expectedMsg)))
				})
			})

			It("syncs pods created in vcluster to host when using a runtimeClass synced from host", func(ctx context.Context) {
				suffix := random.String(6)
				matchingName := "rc-podsync-" + suffix
				podName := "rc-pod-" + suffix

				createRuntimeClass(ctx, matchingName, "runc", map[string]string{"value": "one"})

				By("waiting for the runtimeClass to be synced to vcluster", func() {
					Eventually(func(g Gomega) {
						_, err := vClusterClient.NodeV1().RuntimeClasses().Get(ctx, matchingName, metav1.GetOptions{})
						g.Expect(err).To(Succeed(), "runtimeClass %s not yet synced to vcluster: %v", matchingName, err)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})

				By("creating a pod using the synced runtimeClass in vcluster", func() {
					_, err := vClusterClient.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name:      podName,
							Namespace: "default",
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "nginx", Image: "nginx"},
							},
							RuntimeClassName: &matchingName,
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Pods("default").Delete(ctx, podName, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
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
		})
}
