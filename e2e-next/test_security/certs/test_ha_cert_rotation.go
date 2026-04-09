package certs

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/certs"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/scheme"
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
				hostClient        kubernetes.Interface
				hostRestConfig    *rest.Config
				vClusterName      string
				vClusterNamespace string
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

// execWriteFile writes data to a file inside a container using kubectl exec.
func execWriteFile(ctx context.Context, restConfig *rest.Config, client kubernetes.Interface, namespace, podName, container, filePath string, data []byte) error {
	cmd := []string{"sh", "-c", fmt.Sprintf("cat > %s", filePath)}

	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   cmd,
			Stdin:     true,
			Stdout:    false,
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("creating executor: %w", err)
	}

	reader := bytes.NewReader(data)
	var stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  reader,
		Stderr: &stderr,
	})
	if err != nil {
		return fmt.Errorf("exec failed: %w (stderr: %s)", err, stderr.String())
	}
	return nil
}
