package test_core

import (
	"context"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e/constants"
	"github.com/loft-sh/vcluster/e2e/labels"
	"github.com/loft-sh/vcluster/pkg/util/random"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// EndpointSliceSelectorlessLongNameSpec verifies that a selector-less Service
// with a long name syncs correctly against a real vcluster. A selector-less
// Service is the only case where vcluster sets the kubernetes.io/service-name
// label itself; for a Service with a selector the host endpoint controller
// regenerates the slice and its label. With a 55-byte Service name the naive
// "<svc>-x-<ns>-x-<vc>" label is 74 bytes, which the host apiserver rejects, so
// the slice never syncs. The fix hashes the label to <= 63 bytes while keeping
// it pointed at the host Service.
func EndpointSliceSelectorlessLongNameSpec() {
	Describe("EndpointSlice sync for a selector-less service with a long name",
		labels.Core, labels.Sync,
		func() {
			var (
				hostClient     kubernetes.Interface
				vClusterClient kubernetes.Interface
				vClusterName   string
				hostNS         string
			)

			BeforeEach(func(ctx context.Context) {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterClient = cluster.CurrentKubeClientFrom(ctx)
				Expect(vClusterClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				hostNS = "vcluster-" + vClusterName
			})

			It("syncs the slice with a hashed service-name label that stays within the 63-byte limit", func(ctx context.Context) {
				suffix := random.String(6)
				nsName := "eps-selectorless-" + suffix
				// 55 bytes: a legal virtual Service name whose naive host-side
				// translation would overflow the 63-byte label limit.
				svcName := "frontend-checkout-payments-orders-inventory-billing-svc"
				sliceName := "eps-" + suffix

				By("creating the test namespace in vCluster", func() {
					_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{Name: nsName},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})
				DeferCleanup(func(ctx context.Context) {
					err := vClusterClient.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
					Expect(ctrlclient.IgnoreNotFound(err)).To(Succeed())
				})

				By("creating a selector-less Service (no spec.selector) in vCluster", func() {
					_, err := vClusterClient.CoreV1().Services(nsName).Create(ctx, &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{Name: svcName, Namespace: nsName},
						Spec: corev1.ServiceSpec{
							// no Selector -> selector-less
							Ports: []corev1.ServicePort{{Port: 80, Protocol: corev1.ProtocolTCP}},
						},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				By("creating the EndpointSlice by hand, linked to the service via the service-name label", func() {
					_, err := vClusterClient.DiscoveryV1().EndpointSlices(nsName).Create(ctx, &discoveryv1.EndpointSlice{
						ObjectMeta: metav1.ObjectMeta{
							Name:      sliceName,
							Namespace: nsName,
							Labels: map[string]string{
								translate.K8sServiceNameLabel: svcName,
							},
						},
						AddressType: discoveryv1.AddressTypeIPv4,
						Endpoints:   []discoveryv1.Endpoint{{Addresses: []string{"10.0.0.10"}}},
					}, metav1.CreateOptions{})
					Expect(err).To(Succeed())
				})

				By("asserting the EndpointSlice syncs to the host with a valid, correct service-name label", func() {
					Eventually(func(g Gomega) {
						// The host Service that the virtual selector-less Service maps
						// to, found by annotation (its name carries the vcluster-specific
						// hash, so we read it back rather than recompute it).
						svcs, err := hostClient.CoreV1().Services(hostNS).List(ctx, metav1.ListOptions{})
						g.Expect(err).To(Succeed(), "listing host Services in namespace %s", hostNS)
						var hostSvcName string
						for i := range svcs.Items {
							a := svcs.Items[i].GetAnnotations()
							if a[translate.NameAnnotation] == svcName && a[translate.NamespaceAnnotation] == nsName {
								hostSvcName = svcs.Items[i].Name
								break
							}
						}
						g.Expect(hostSvcName).NotTo(BeEmpty(), "host Service for the virtual selector-less Service not found")

						slices, err := hostClient.DiscoveryV1().EndpointSlices(hostNS).List(ctx, metav1.ListOptions{})
						g.Expect(err).To(Succeed(), "listing host EndpointSlices in namespace %s", hostNS)

						var hostSlice *discoveryv1.EndpointSlice
						for i := range slices.Items {
							a := slices.Items[i].GetAnnotations()
							if a[translate.NameAnnotation] == sliceName && a[translate.NamespaceAnnotation] == nsName {
								hostSlice = &slices.Items[i]
								break
							}
						}
						// Pre-fix, the host apiserver rejects the >63-byte label,
						// so the slice never appears and this stays nil -> timeout.
						g.Expect(hostSlice).NotTo(BeNil(), "synced host EndpointSlice not found; a >63-byte service-name label would have been rejected by the host apiserver")

						label := hostSlice.GetLabels()[translate.K8sServiceNameLabel]
						g.Expect(len(label)).To(BeNumerically("<=", 63), "service-name label exceeds the 63-byte limit: %q", label)
						g.Expect(label).To(Equal(hostSvcName), "service-name label must point at the host Service name")
					}).WithContext(ctx).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutLong).Should(Succeed())
				})
			})
		})
}
