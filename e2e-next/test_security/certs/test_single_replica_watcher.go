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

// SingleReplicaWatcherSpec verifies that the cert watcher works correctly in
// a single-replica deployment where no lease coordination is needed. This is
// the default deployment mode where coordination.k8s.io/leases RBAC may not
// be granted, so the watcher must skip lease acquisition and rotate directly.
//
// Must be called inside a Describe that has cluster.Use() for the vcluster and host cluster.
func SingleReplicaWatcherSpec() {
	Describe("Single-replica cert watcher rotation",
		Ordered,
		labels.Security,
		func() {
			var (
				hostClient         kubernetes.Interface
				hostRestConfig     *rest.Config
				vClusterName       string
				vClusterNamespace  string
				controlPlaneUIDs   map[string]struct{}
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

			It("should have the vCluster pod running and ready", func(ctx context.Context) {
				waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutVeryLong)

				controlPlaneUIDs = podUIDs(listPodsBySelector(ctx, hostClient, vClusterNamespace, "app=vcluster,release="+vClusterName))
				Expect(controlPlaneUIDs).To(HaveLen(1), "expected exactly 1 pod for single-replica")

				By("Recording the original CA cert NotAfter", func() {
					secret, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
						certs.CertSecretName(vClusterName), metav1.GetOptions{})
					Expect(err).To(Succeed())
					ca, err := parseCertFromPEM(secret.Data["ca.crt"])
					Expect(err).To(Succeed())
					originalCANotAfter = ca.NotAfter
				})
			})

			// Spec 2 depends on 1: write an expiring cert to disk inside the
			// running pod so the watcher detects it on its next check (every 15s).
			It("should inject an expiring cert into the running pod", func(ctx context.Context) {
				expiringCertPEM := generateExpiringCertPEM(30 * 24 * time.Hour)

				By("Writing expiring apiserver.crt to disk in the pod", func() {
					pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
						LabelSelector: "app=vcluster,release=" + vClusterName,
					})
					Expect(err).To(Succeed())
					Expect(pods.Items).To(HaveLen(1), "expected exactly 1 pod for single-replica")

					err = execWriteFile(ctx, hostRestConfig, hostClient,
						vClusterNamespace, pods.Items[0].Name, "syncer",
						"/data/pki/apiserver.crt", expiringCertPEM)
					Expect(err).To(Succeed(), "failed to write expiring cert to pod")
				})
			})

			// Spec 3 depends on 2: the watcher should detect the expiring cert,
			// rotate without lease coordination, and trigger a rollout of the
			// single control-plane workload.
			It("should patch the control-plane rollout annotation without lease coordination", func(ctx context.Context) {
				By("Waiting for the control-plane workload to be marked for rollout", func() {
					Eventually(func(g Gomega) {
						rolloutAt, err := getControlPlaneRolloutAnnotation(ctx, hostClient, vClusterNamespace, vClusterName)
						g.Expect(err).To(Succeed())
						g.Expect(rolloutAt).NotTo(BeEmpty(),
							"single-replica workload should have the cert rotation annotation")
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
				})
			})

			// Spec 4 depends on 3: after the rollout starts, the secret should be
			// renewed and the old pod should be replaced by the controller.
			It("should rotate certs and roll the single replica", func(ctx context.Context) {
				By("Waiting for the secret to contain a renewed apiserver cert", func() {
					expectCertRenewed(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutVeryLong)
				})

				By("Waiting for the old pod to be replaced and the new pod to be ready", func() {
					waitForPodsRolled(ctx, hostClient, vClusterNamespace, "app=vcluster,release="+vClusterName, controlPlaneUIDs, 1, constants.PollingTimeoutVeryLong)
				})
			})

			// Spec 5 depends on 4: verify CA was not rotated.
			It("should preserve the CA cert during watcher rotation", func(ctx context.Context) {
				expectCAPreserved(ctx, hostClient, vClusterNamespace, vClusterName, originalCANotAfter)
			})

			// Spec 6 depends on 4: verify all leaf certs are renewed.
			It("should have all leaf certs valid after watcher rotation", func(ctx context.Context) {
				expectAllLeafCertsRenewed(ctx, hostClient, vClusterNamespace, vClusterName)
			})

			// Spec 7 depends on 4: verify no rotation lease was created, confirming
			// the single-replica path skipped lease coordination.
			It("should not have created a rotation lease", func(ctx context.Context) {
				leaseName := translate.SafeConcatName("vcluster", vClusterName, "cert-rotation")
				_, err := hostClient.CoordinationV1().Leases(vClusterNamespace).Get(ctx,
					leaseName, metav1.GetOptions{})
				Expect(kerrors.IsNotFound(err) || kerrors.IsForbidden(err)).To(BeTrue(),
					"rotation lease should not exist for single-replica deployment, got: %v", err)
			})

			// Spec 8 depends on 4: verify no deployed etcd rollout was attempted.
			// This cluster uses embedded etcd (no deploy.enabled), so the config
			// guard should have skipped the etcd StatefulSet rollout entirely.
			It("should not have attempted a deployed etcd rollout", func(ctx context.Context) {
				etcdName := vClusterName + "-etcd"
				_, err := hostClient.AppsV1().StatefulSets(vClusterNamespace).Get(ctx,
					etcdName, metav1.GetOptions{})
				Expect(kerrors.IsNotFound(err)).To(BeTrue(),
					"etcd StatefulSet should not exist for embedded etcd deployment, got: %v", err)
			})
		},
	)
}
