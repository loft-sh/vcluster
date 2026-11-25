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
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/kubernetes"
)

// HostNameCollisionSpec verifies that two virtual services whose names translate to the same
// host name can't touch each other's host service. prod-x-<team>/web and <team>/web-x-prod both
// translate to web-x-prod-x-<team>-x-<vcluster>, so the second service, created with server-side
// apply, must stay unsynced with a SyncError event instead of taking over or deleting the host
// service of the first, and must not change the type of that host service.
func HostNameCollisionSpec() {
	Describe("Host name collision between virtual objects",
		labels.Core, labels.Sync, labels.Events,
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

			It("should not let a colliding service take over or delete another namespace's host service", labels.PR, func(ctx context.Context) {
				team := "team-" + random.String(6)
				victimNS, victimName := "prod-x-"+team, "web"
				attackerNS, attackerName := team, "web-x-prod"
				hostNS := vClusterHostNamespace(vClusterName)
				hostName := translate.SingleNamespaceHostName(victimName, victimNS, vClusterName)
				Expect(translate.SingleNamespaceHostName(attackerName, attackerNS, vClusterName)).To(Equal(hostName),
					"test names must translate to the same host name")

				for _, ns := range []string{victimNS, attackerNS} {
					By("creating namespace "+ns+" in the virtual cluster", func() {
						_, err := vClusterClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
							ObjectMeta: metav1.ObjectMeta{Name: ns},
						}, metav1.CreateOptions{})
						Expect(err).To(Succeed())
						DeferCleanup(func(ctx context.Context) {
							err := vClusterClient.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{})
							if !kerrors.IsNotFound(err) {
								Expect(err).To(Succeed())
							}
						})
					})
				}

				expectHostServiceOfVictim := func(g Gomega, ctx context.Context) {
					hostSvc, err := hostClient.CoreV1().Services(hostNS).Get(ctx, hostName, metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred(), "host service %s/%s of %s/%s should exist", hostNS, hostName, victimNS, victimName)
					g.Expect(hostSvc.Annotations).To(HaveKeyWithValue(translate.NamespaceAnnotation, victimNS))
					g.Expect(hostSvc.Annotations).To(HaveKeyWithValue(translate.NameAnnotation, victimName))
					g.Expect(hostSvc.Spec.Ports).To(HaveLen(1))
					g.Expect(hostSvc.Spec.Ports[0].Port).To(Equal(int32(80)))
				}

				By("creating the first service and waiting for its host service", func() {
					_, err := vClusterClient.CoreV1().Services(victimNS).Create(ctx, collisionService(victimName, 80), metav1.CreateOptions{})
					Expect(err).To(Succeed())

					Eventually(expectHostServiceOfVictim).WithContext(ctx).
						WithPolling(constants.PollingInterval).
						WithTimeout(constants.PollingTimeout).
						Should(Succeed())
				})

				By("verifying a plain create of the colliding service is rejected", func() {
					// the service create filter (pkg/server/filters/service.go) creates the host service first
					_, err := vClusterClient.CoreV1().Services(attackerNS).Create(ctx, collisionService(attackerName, 9999), metav1.CreateOptions{})
					Expect(kerrors.IsAlreadyExists(err)).To(BeTrue(), "expected AlreadyExists, got %v", err)
				})

				By("creating the colliding service with server-side apply", func() {
					// server-side apply is a patch, so it bypasses the service create filter and leaves the collision to the syncer
					_, err := vClusterClient.CoreV1().Services(attackerNS).Apply(ctx,
						corev1apply.Service(attackerName, attackerNS).WithSpec(corev1apply.ServiceSpec().
							WithType(corev1.ServiceTypeExternalName).
							WithExternalName("example.com").
							WithPorts(corev1apply.ServicePort().WithName("http").WithPort(9999).WithProtocol(corev1.ProtocolTCP)),
						),
						metav1.ApplyOptions{FieldManager: "e2e-host-name-collision"},
					)
					Expect(err).To(Succeed())
				})

				By("waiting for a SyncError event on the colliding service that doesn't reveal the host name", func() {
					Eventually(func(g Gomega, ctx context.Context) {
						eventList, err := vClusterClient.CoreV1().Events(attackerNS).List(ctx, metav1.ListOptions{
							FieldSelector: "involvedObject.name=" + attackerName,
						})
						g.Expect(err).NotTo(HaveOccurred())

						syncErrorCount := 0
						for _, event := range eventList.Items {
							if event.Reason != "SyncError" {
								continue
							}
							syncErrorCount++
							g.Expect(event.Type).To(Equal(corev1.EventTypeWarning))
							g.Expect(event.Message).NotTo(ContainSubstring(hostName),
								"SyncError event message %q must not contain the host name %q", event.Message, hostName)
						}
						g.Expect(syncErrorCount).To(BeNumerically(">", 0), "expected a SyncError event on %s/%s", attackerNS, attackerName)
					}).WithContext(ctx).
						WithPolling(constants.PollingInterval).
						WithTimeout(constants.PollingTimeout).
						Should(Succeed())
				})

				By("verifying the host service still belongs to the first service", func() {
					Consistently(expectHostServiceOfVictim).WithContext(ctx).
						WithPolling(constants.PollingInterval).
						WithTimeout(constants.PollingTimeoutVeryShort).
						Should(Succeed())
				})

				By("verifying the colliding service can't switch its type from ExternalName", func() {
					// the service update filter (pkg/server/filters/service.go) changes the type of the host service first
					svc, err := vClusterClient.CoreV1().Services(attackerNS).Get(ctx, attackerName, metav1.GetOptions{})
					Expect(err).To(Succeed())
					svc.Spec.Type = corev1.ServiceTypeClusterIP
					svc.Spec.ExternalName = ""
					svc.Spec.ClusterIP = ""
					_, err = vClusterClient.CoreV1().Services(attackerNS).Update(ctx, svc, metav1.UpdateOptions{})
					Expect(kerrors.IsConflict(err)).To(BeTrue(), "expected Conflict, got %v", err)
					Expect(err.Error()).NotTo(ContainSubstring(hostName), "error %q must not contain the host name %q", err.Error(), hostName)

					Consistently(expectHostServiceOfVictim).WithContext(ctx).
						WithPolling(constants.PollingInterval).
						WithTimeout(constants.PollingTimeoutVeryShort).
						Should(Succeed())
				})

				By("deleting the colliding service", func() {
					err := vClusterClient.CoreV1().Services(attackerNS).Delete(ctx, attackerName, metav1.DeleteOptions{})
					Expect(err).To(Succeed())
				})

				By("verifying the host service of the first service was not deleted", func() {
					Consistently(expectHostServiceOfVictim).WithContext(ctx).
						WithPolling(constants.PollingInterval).
						WithTimeout(constants.PollingTimeoutVeryShort).
						Should(Succeed())
				})
			})
		},
	)
}

func collisionService(name string, port int32) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{Name: "http", Port: port, Protocol: corev1.ProtocolTCP}},
		},
	}
}
