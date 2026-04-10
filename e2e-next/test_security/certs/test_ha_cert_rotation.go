package certs

import (
	"context"
	"time"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// HACertRotationSpec verifies that HA cert rotation is coordinated via a
// Lease so that replicas don't all restart simultaneously.
//
// This test uses a 2-replica vcluster with short-lived certs (3m) and a
// short watcher check interval (15s). After pods are running, we write an
// expiring cert directly to disk inside each pod (bypassing the startup
// EnsureCerts check). The watcher detects the expiring cert on its next
// check and the first replica to acquire the rotation lease performs the
// rotation.
//
// Must be called inside a Describe that has cluster.Use() for the vcluster and host cluster.
func HACertRotationSpec() {
	Describe("HA coordinated cert rotation",
		Ordered,
		labels.Core,
		labels.Security,
		func() {
			var (
				hostClient         kubernetes.Interface
				hostRestConfig     *rest.Config
				vClusterName       string
				vClusterNamespace  string
				controlPlaneUIDs   map[string]struct{}
				etcdUIDs           map[string]struct{}
				originalCANotAfter time.Time
			)

			BeforeAll(func(ctx context.Context) context.Context {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				hostRestConfig = cluster.From(ctx, constants.GetHostClusterName()).KubernetesRestConfig()
				Expect(hostRestConfig).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterNamespace = "vcluster-" + vClusterName
				return ctx
			})

			// Spec 1: both replicas are running. Capture the original pod UIDs so
			// we can later verify that the rollout replaced every replica.
			It("should have all HA vCluster pods running and ready", func(ctx context.Context) {
				By("Waiting for all replicas to be ready", func() {
					Eventually(func(g Gomega) {
						pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
							LabelSelector: "app=vcluster,release=" + vClusterName,
						})
						g.Expect(err).To(Succeed())
						g.Expect(len(pods.Items)).To(BeNumerically(">=", 2),
							"expected at least 2 vcluster pods for HA, got %d", len(pods.Items))

						for _, pod := range pods.Items {
							for _, container := range pod.Status.ContainerStatuses {
								g.Expect(container.State.Running).NotTo(BeNil(),
									"container %s in pod %s should be running", container.Name, pod.Name)
								g.Expect(container.Ready).To(BeTrue(),
									"container %s in pod %s should be ready", container.Name, pod.Name)
							}
						}
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
				})

				By("Capturing the original control-plane and etcd pod identities", func() {
					controlPlaneUIDs = podUIDs(listPodsBySelector(ctx, hostClient, vClusterNamespace, "app=vcluster,release="+vClusterName))
					etcdUIDs = podUIDs(listPodsBySelector(ctx, hostClient, vClusterNamespace, "app=vcluster-etcd,release="+vClusterName))
					Expect(controlPlaneUIDs).To(HaveLen(2), "expected 2 control-plane pods before rollout")
					Expect(etcdUIDs).NotTo(BeEmpty(), "expected deployed etcd pod(s) before rollout")
				})

				By("Recording the original CA cert NotAfter", func() {
					secret, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
						certs.CertSecretName(vClusterName), metav1.GetOptions{})
					Expect(err).To(Succeed())
					ca, err := parseCertFromPEM(secret.Data["ca.crt"])
					Expect(err).To(Succeed())
					originalCANotAfter = ca.NotAfter
				})
			})

			// Spec 2 depends on 1: write an expiring cert directly to disk inside
			// each running pod. This bypasses the startup EnsureCerts check (which
			// already ran with valid certs) so the runtime watcher is the one that
			// detects the expiry.
			It("should inject expiring certs into running pods", func(ctx context.Context) {
				expiringCertPEM := generateExpiringCertPEM(30 * 24 * time.Hour)

				By("Writing expiring apiserver.crt to disk in each pod", func() {
					pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
						LabelSelector: "app=vcluster,release=" + vClusterName,
					})
					Expect(err).To(Succeed())
					Expect(pods.Items).NotTo(BeEmpty())

					for _, pod := range pods.Items {
						err := execWriteFile(ctx, hostRestConfig, hostClient,
							vClusterNamespace, pod.Name, "syncer",
							"/data/pki/apiserver.crt", expiringCertPEM)
						Expect(err).To(Succeed(),
							"failed to write expiring cert to pod %s: %v", pod.Name, err)
					}
				})
			})

			// Spec 3 depends on 2: wait for the cert rotation lease to be created.
			// The watcher checks every 15s. When it detects the expiring cert on
			// disk, the first replica to acquire the lease performs the rotation.
			It("should create a cert rotation lease for coordination", func(ctx context.Context) {
				By("Waiting for the rotation lease to appear", func() {
					leaseName := translate.SafeConcatName("vcluster", vClusterName, "cert-rotation")
					Eventually(func(g Gomega) {
						lease, err := hostClient.CoordinationV1().Leases(vClusterNamespace).Get(ctx,
							leaseName, metav1.GetOptions{})
						g.Expect(err).To(Succeed(), "cert rotation lease should exist")
						g.Expect(lease.Spec.HolderIdentity).NotTo(BeNil(),
							"lease should have a holder identity")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
				})
			})

			// Spec 4 depends on 3: after the watcher rotates certs, it should patch
			// the workload templates to trigger rollout propagation.
			It("should patch rollout annotations on the workloads", func(ctx context.Context) {
				var controlPlaneRolloutAt string

				By("Waiting for the control-plane template annotation to be set", func() {
					Eventually(func(g Gomega) {
						var err error
						controlPlaneRolloutAt, err = getControlPlaneRolloutAnnotation(ctx, hostClient, vClusterNamespace, vClusterName)
						g.Expect(err).To(Succeed())
						g.Expect(controlPlaneRolloutAt).NotTo(BeEmpty(),
							"control-plane workload should have the cert rotation annotation")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
				})

				By("Waiting for the deployed etcd template annotation to match", func() {
					Eventually(func(g Gomega) {
						etcdRolloutAt, err := getStatefulSetRolloutAnnotation(ctx, hostClient, vClusterNamespace, vClusterName+"-etcd")
						g.Expect(err).To(Succeed())
						g.Expect(etcdRolloutAt).To(Equal(controlPlaneRolloutAt),
							"etcd and control-plane should use the same rollout marker")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
				})
			})

			// Spec 5 depends on 4: verify the rollout replaced every pod, not just
			// the lease holder, and that the secret contains renewed certs.
			It("should roll all HA replicas and renew the certs", func(ctx context.Context) {
				By("Waiting for the control-plane rollout to replace every pod", func() {
					waitForPodsRolled(ctx, hostClient, vClusterNamespace, "app=vcluster,release="+vClusterName, controlPlaneUIDs, 2, constants.PollingTimeoutVeryLong)
				})

				By("Waiting for the deployed etcd rollout to replace every pod", func() {
					waitForPodsRolled(ctx, hostClient, vClusterNamespace, "app=vcluster-etcd,release="+vClusterName, etcdUIDs, len(etcdUIDs), constants.PollingTimeoutVeryLong)
				})

				By("Verifying the apiserver cert was renewed in the secret", func() {
					expectCertRenewed(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutVeryLong)
				})
			})

			// Spec 6 depends on 5: verify CA was not rotated.
			It("should preserve the CA cert during HA watcher rotation", func(ctx context.Context) {
				expectCAPreserved(ctx, hostClient, vClusterNamespace, vClusterName, originalCANotAfter)
			})

			// Spec 7 depends on 5: verify all leaf certs including etcd certs are renewed.
			It("should have all leaf certs valid after HA watcher rotation", func(ctx context.Context) {
				expectAllLeafCertsRenewed(ctx, hostClient, vClusterNamespace, vClusterName)
			})

			// Spec 8: cleanup the lease
			It("should clean up the cert rotation lease", func(ctx context.Context) {
				leaseName := translate.SafeConcatName("vcluster", vClusterName, "cert-rotation")
				err := hostClient.CoordinationV1().Leases(vClusterNamespace).Delete(ctx,
					leaseName, metav1.DeleteOptions{})
				if !kerrors.IsNotFound(err) {
					Expect(err).To(Succeed())
				}
			})
		},
	)
}
