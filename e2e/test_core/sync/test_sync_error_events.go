package test_core

import (
	"context"
	"fmt"
	"sync"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e/constants"
	"github.com/loft-sh/vcluster/e2e/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SyncErrorSanitisationSpec verifies that SyncError warning events recorded on
// virtual pods never expose the host-side translated pod name.
//
// The host pod name (e.g. "my-pod-x-default-x-my-vc") must not appear in event
// messages visible from inside the virtual cluster. The sanitising event recorder
// (pkg/syncer/translator/sanitising_event_recorder.go) rewrites it back to the
// virtual name before the event is persisted.
func SyncErrorSanitisationSpec() {
	Describe("SyncError event sanitisation",
		labels.Core, labels.Events, labels.Pods, labels.Sync,
		func() {
			var (
				hostClient     kubernetes.Interface
				vClusterClient kubernetes.Interface
				vClusterName   string
			)

			BeforeEach(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
			})

			It("should not expose the host pod name in SyncError events", labels.PR, func(ctx context.Context) {
				suffix := random.String(6)
				ns := "sync-err-test-" + suffix
				podName := "test-pod-" + suffix

				By("creating a test namespace in the virtual cluster", func() {
					_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: ns},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{})
					if !kerrors.IsNotFound(err) {
						Expect(err).To(Succeed())
					}
				})
				// hostPodName is the translated host-side name that MUST NOT appear
				// in any virtual pod event (e.g. "test-pod-abc123-x-sync-err-test-abc123-x-my-vc").
				hostPodName := translate.SingleNamespaceHostName(podName, ns, vClusterName)
				hostNS := vClusterHostNamespace(vClusterName)

				By("creating a pod in the virtual cluster", func() {
					_, err := vClusterClient.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name:      podName,
							Namespace: ns,
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            testingContainerName,
									Image:           testingContainerImage,
									ImagePullPolicy: corev1.PullIfNotPresent,
									SecurityContext: &corev1.SecurityContext{
										Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
										RunAsNonRoot:             boolPtr(true),
										RunAsUser:                int64Ptr(12345),
										AllowPrivilegeEscalation: boolPtr(false),
										SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
									},
								},
							},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
					DeferCleanup(func(ctx context.Context) {
						err := vClusterClient.CoreV1().Pods(ns).Delete(ctx, podName, metav1.DeleteOptions{})
						if !kerrors.IsNotFound(err) {
							Expect(err).To(Succeed())
						}
					})
				})

				By("waiting for the syncer to create the host pod", func() {
					Eventually(func(g Gomega, ctx context.Context) {
						_, err := hostClient.CoreV1().Pods(hostNS).Get(ctx, hostPodName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(),
							"host pod %s/%s should exist after sync", hostNS, hostPodName)
					}).WithContext(ctx).
						WithPolling(constants.PollingInterval).
						WithTimeout(constants.PollingTimeout).
						Should(Succeed())
				})

				// Race the syncer against continuous host and virtual pod updates.
				//
				// The pod syncer fetches the host pod once at the start of each reconcile.
				// If the host pod's resource version changes before the syncer's patch lands,
				// the API server returns a 409 Conflict; the pod syncer records a SyncError
				// event (pkg/controllers/resources/pods/syncer.go) before the conflict is
				// silently requeued by the outer controller.
				//
				// Both goroutines keep mutating for the entire verification window below,
				// rather than running a fixed number of iterations up front. A single short
				// burst can finish before the syncer ever loses the race, after which the
				// syncer settles and no further conflicts occur — that is what made this test
				// flaky. Keeping the race window open until a SyncError event is observed
				// makes the conflict reliable. Errors are intentionally ignored: an update
				// that fails because the syncer just changed the pod is exactly the
				// contention we want.
				stop := make(chan struct{})
				var wg sync.WaitGroup

				runMutator := func(bump func(i int) error) {
					defer wg.Done()
					for i := 0; ; i++ {
						select {
						case <-stop:
							return
						case <-ctx.Done():
							return
						default:
						}
						_ = bump(i)
					}
				}

				By("continuously racing host and virtual pod updates against the syncer to force SyncError events", func() {
					wg.Add(2)
					go runMutator(func(i int) error {
						hp, err := hostClient.CoreV1().Pods(hostNS).Get(ctx, hostPodName, metav1.GetOptions{})
						if err != nil {
							return err
						}
						if hp.Annotations == nil {
							hp.Annotations = map[string]string{}
						}
						hp.Annotations["race-bump"] = fmt.Sprintf("%d", i)
						_, err = hostClient.CoreV1().Pods(hostNS).Update(ctx, hp, metav1.UpdateOptions{})
						return err
					})
					go runMutator(func(i int) error {
						vPod, err := vClusterClient.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
						if err != nil {
							return err
						}
						if vPod.Annotations == nil {
							vPod.Annotations = map[string]string{}
						}
						vPod.Annotations["reconcile-trigger"] = fmt.Sprintf("%d", i)
						_, err = vClusterClient.CoreV1().Pods(ns).Update(ctx, vPod, metav1.UpdateOptions{})
						return err
					})
				})

				// Stop the background mutators before the pod/namespace cleanups run.
				// DeferCleanup is LIFO, so this runs before the earlier-registered
				// pod and namespace deletions.
				DeferCleanup(func() {
					close(stop)
					wg.Wait()
				})

				By("verifying at least one SyncError event was recorded without the host pod name", func() {
					// The Eventually here serves two purposes:
					// 1. It waits until the syncer has produced at least one SyncError event,
					//    confirming that the sanitiser code path was actually exercised.
					// 2. It fails immediately if any SyncError message contains the host pod
					//    name, catching a regression in the sanitising recorder.
					//
					// If no SyncError events are produced within PollingTimeout the test fails
					// — it does NOT pass vacuously when events are absent.
					Eventually(func(g Gomega, ctx context.Context) {
						eventList, err := vClusterClient.CoreV1().Events(ns).List(ctx, metav1.ListOptions{
							FieldSelector: "involvedObject.name=" + podName,
						})
						g.Expect(err).NotTo(HaveOccurred())

						syncErrorCount := 0
						for _, event := range eventList.Items {
							if event.Reason != "SyncError" {
								continue
							}
							syncErrorCount++
							g.Expect(event.Message).NotTo(ContainSubstring(hostPodName),
								"SyncError event message %q must not contain the host pod name %q",
								event.Message, hostPodName)
						}
						g.Expect(syncErrorCount).To(BeNumerically(">", 0),
							"expected at least one SyncError event to verify the sanitiser ran")
					}).WithContext(ctx).
						WithPolling(constants.PollingInterval).
						WithTimeout(constants.PollingTimeout).
						Should(Succeed())
				})
			})
		},
	)
}
