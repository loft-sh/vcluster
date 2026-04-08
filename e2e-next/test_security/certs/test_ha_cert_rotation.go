package certs

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"time"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// HACertRotationSpec verifies that HA cert rotation is coordinated via a
// Lease so that replicas don't all restart simultaneously.
//
// This test uses a 2-replica vcluster with short-lived certs (3m) and a
// short watcher check interval (15s). The watcher detects expiring certs
// and the first replica to acquire the rotation lease performs the rotation.
//
// Must be called inside a Describe that has cluster.Use() for the vcluster and host cluster.
func HACertRotationSpec() {
	Describe("HA coordinated cert rotation",
		Ordered,
		labels.Core,
		labels.Security,
		func() {
			var (
				hostClient        kubernetes.Interface
				vClusterName      string
				vClusterNamespace string
			)

			BeforeAll(func(ctx context.Context) context.Context {
				hostClient = cluster.KubeClientFrom(ctx, constants.GetHostClusterName())
				Expect(hostClient).NotTo(BeNil())
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterNamespace = "vcluster-" + vClusterName
				return ctx
			})

			// Spec 1: both replicas are running
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
			})

			// Spec 2 depends on 1: inject expiring certs into the secret so the
			// watcher detects them on its next check (every 15s).
			It("should inject expiring leaf certs into the secret", func(ctx context.Context) {
				By("Patching the apiserver.crt to expire in 30 days", func() {
					secret, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
						certs.CertSecretName(vClusterName), metav1.GetOptions{})
					Expect(err).To(Succeed())

					expiringCert := generateExpiringCertPEM(30 * 24 * time.Hour)
					secret.Data["apiserver.crt"] = expiringCert

					_, err = hostClient.CoreV1().Secrets(vClusterNamespace).Update(ctx,
						secret, metav1.UpdateOptions{})
					Expect(err).To(Succeed())
				})

				By("Restarting all pods so they download the expiring cert to disk", func() {
					pods, err := hostClient.CoreV1().Pods(vClusterNamespace).List(ctx, metav1.ListOptions{
						LabelSelector: "app=vcluster,release=" + vClusterName,
					})
					Expect(err).To(Succeed())

					for _, pod := range pods.Items {
						err := hostClient.CoreV1().Pods(vClusterNamespace).Delete(ctx,
							pod.Name, metav1.DeleteOptions{})
						Expect(err).To(Succeed())
					}
				})

				By("Waiting for pods to come back ready", func() {
					waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutVeryLong)
				})
			})

			// Spec 3 depends on 2: wait for the cert rotation lease to be created.
			// The watcher checks every 15s. When it detects expiring certs, the
			// first replica to acquire the lease performs the rotation.
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

			// Spec 4 depends on 3: after the watcher rotates and the pod restarts,
			// verify certs are renewed and the cluster is healthy.
			It("should have renewed certs after coordinated rotation", func(ctx context.Context) {
				By("Waiting for all pods to be ready after watcher-triggered restart", func() {
					waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutVeryLong)
				})

				By("Verifying the apiserver cert was renewed", func() {
					Eventually(func(g Gomega) {
						secret, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
							certs.CertSecretName(vClusterName), metav1.GetOptions{})
						g.Expect(err).To(Succeed())

						block, _ := pem.Decode(secret.Data["apiserver.crt"])
						g.Expect(block).NotTo(BeNil(), "failed to decode apiserver cert PEM")

						cert, err := x509.ParseCertificate(block.Bytes)
						g.Expect(err).To(Succeed())

						g.Expect(cert.NotAfter.After(time.Now().Add(90*24*time.Hour))).To(BeTrue(),
							"apiserver cert should have been renewed, NotAfter=%s", cert.NotAfter.Format(time.RFC3339))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
				})
			})

			// Spec 5: cleanup the lease
			It("should clean up the cert rotation lease", func(ctx context.Context) {
				leaseName := translate.SafeConcatName("vcluster", vClusterName, "cert-rotation")
				err := hostClient.CoordinationV1().Leases(vClusterNamespace).Delete(ctx,
					leaseName, metav1.DeleteOptions{})
				if err != nil {
					klog.Infof("Lease cleanup: %v", err)
				}
			})
		},
	)
}
