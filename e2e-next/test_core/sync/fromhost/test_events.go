package fromhost

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
)

var _ = Describe("Events force-sync from host via annotation",
	labels.Core,
	labels.Sync,
	labels.Events,
	cluster.Use(clusters.K8sDefaultEndpointVCluster),
	cluster.Use(clusters.HostCluster),
	func() {
		var (
			hostClient        kubernetes.Interface
			vClusterClient    kubernetes.Interface
			vClusterName      = clusters.K8sDefaultEndpointVClusterName
			vClusterNamespace = "vcluster-" + vClusterName
		)

		BeforeEach(func(ctx context.Context) {
			hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
			Expect(hostClient).NotTo(BeNil())
			vClusterClient = cluster.CurrentKubeClientFrom(ctx)
			Expect(vClusterClient).NotTo(BeNil())
		})

		It("should sync a force-synced event from host to vCluster", func(ctx context.Context) {
			suffix := random.String(6)
			eventName := "force-sync-event-" + suffix
			eventMessage := "test message for force-sync"
			cmName := "dummy-cm-for-event-" + suffix

			// Create ConfigMap as involved object
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cmName,
					Namespace: vClusterNamespace,
				},
			}
			_, err := hostClient.CoreV1().ConfigMaps(vClusterNamespace).Create(ctx, cm, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				err := hostClient.CoreV1().ConfigMaps(vClusterNamespace).Delete(ctx, cmName, metav1.DeleteOptions{})
				if !kerrors.IsNotFound(err) {
					Expect(err).To(Succeed())
				}
			})

			// Build event reference from the ConfigMap
			involvedObj, err := hostClient.CoreV1().ConfigMaps(vClusterNamespace).Get(ctx, cmName, metav1.GetOptions{})
			Expect(err).To(Succeed())

			ref, err := reference.GetReference(runtime.NewScheme(), involvedObj)
			if err != nil {
				ref, err = reference.GetReference(scheme.Scheme, involvedObj)
				Expect(err).To(Succeed())
			}

			// Create force-synced event
			event := &corev1.Event{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventName,
					Namespace: vClusterNamespace,
					Annotations: map[string]string{
						"vcluster.loft.sh/force-sync": "true",
					},
				},
				InvolvedObject: *ref,
				Message:        eventMessage,
			}
			_, err = hostClient.CoreV1().Events(vClusterNamespace).Create(ctx, event, metav1.CreateOptions{})
			Expect(err).To(Succeed())
			DeferCleanup(func(ctx context.Context) {
				err := hostClient.CoreV1().Events(vClusterNamespace).Delete(ctx, eventName, metav1.DeleteOptions{})
				if !kerrors.IsNotFound(err) {
					Expect(err).To(Succeed())
				}
			})

			By("Waiting for the event to appear in the vCluster", func() {
				Eventually(func(g Gomega) {
					virtualEvent, err := vClusterClient.CoreV1().Events("default").Get(ctx, eventName, metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred(),
						"event %s not yet synced to vCluster default namespace", eventName)
					g.Expect(virtualEvent.Message).To(Equal(eventMessage),
						"event message mismatch: expected %q, got %q", eventMessage, virtualEvent.Message)
				}).
					WithPolling(constants.PollingInterval).
					WithTimeout(constants.PollingTimeout).
					Should(Succeed())
			})
		})
	},
)
