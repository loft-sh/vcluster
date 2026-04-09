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
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
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

			It("should have the vCluster pod running and ready", func(ctx context.Context) {
				waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutVeryLong)
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

					err = execWriteFileSingleReplica(ctx, hostRestConfig, hostClient,
						vClusterNamespace, pods.Items[0].Name, "syncer",
						"/data/pki/apiserver.crt", expiringCertPEM)
					Expect(err).To(Succeed(), "failed to write expiring cert to pod")
				})
			})

			// Spec 3 depends on 2: the watcher should detect the expiring cert,
			// rotate without lease coordination, and trigger a graceful restart.
			// After the pod restarts, certs should be renewed.
			It("should rotate certs and restart without lease coordination", func(ctx context.Context) {
				By("Waiting for the pod to restart with renewed certs", func() {
					Eventually(func(g Gomega) {
						secret, err := hostClient.CoreV1().Secrets(vClusterNamespace).Get(ctx,
							certs.CertSecretName(vClusterName), metav1.GetOptions{})
						g.Expect(err).To(Succeed())

						block, _ := pem.Decode(secret.Data["apiserver.crt"])
						g.Expect(block).NotTo(BeNil(), "failed to decode apiserver cert PEM")

						cert, err := x509.ParseCertificate(block.Bytes)
						g.Expect(err).To(Succeed())

						g.Expect(cert.NotAfter.After(time.Now().Add(90*24*time.Hour))).To(BeTrue(),
							"apiserver cert should have been renewed by the watcher, NotAfter=%s",
							cert.NotAfter.Format(time.RFC3339))
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutVeryLong).Should(Succeed())
				})

				By("Waiting for the pod to be ready after restart", func() {
					waitForPodsReady(ctx, hostClient, vClusterNamespace, vClusterName, constants.PollingTimeoutVeryLong)
				})
			})

			// Spec 4 depends on 3: verify no rotation lease was created, confirming
			// the single-replica path skipped lease coordination.
			It("should not have created a rotation lease", func(ctx context.Context) {
				leaseName := translate.SafeConcatName("vcluster", vClusterName, "cert-rotation")
				_, err := hostClient.CoordinationV1().Leases(vClusterNamespace).Get(ctx,
					leaseName, metav1.GetOptions{})
				Expect(kerrors.IsNotFound(err) || kerrors.IsForbidden(err)).To(BeTrue(),
					"rotation lease should not exist for single-replica deployment, got: %v", err)
			})
		},
	)
}

// execWriteFileSingleReplica writes data to a file inside a container using kubectl exec.
func execWriteFileSingleReplica(ctx context.Context, restConfig *rest.Config, client kubernetes.Interface, namespace, podName, container, filePath string, data []byte) error {
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
