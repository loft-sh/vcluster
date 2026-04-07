// Package lifecycle contains vCluster CLI lifecycle tests (connect, pause/resume, etc.).
package lifecycle

import (
	"context"
	"os/exec"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// ConnectSpec registers vcluster connect tests.
// All tests shell out to the vcluster binary (must be in $PATH).
// Tests that produce kubeconfig use --print to avoid blocking on port-forwarding.
//
// Migrated from test/e2e_cli/connect/connect.go. Changes from old suite:
//   - Old "connect by name" and "connect --print to file" merged into one test
//     ("print kubeconfig and use it") because both validate the same thing:
//     connect produces a usable kubeconfig. The merged test also verifies the
//     kubeconfig works by listing pods.
//   - Old "unreachable server" test removed: --print with --server just embeds
//     the server in the kubeconfig without actually connecting, so the error
//     case cannot be tested via --print.
//   - All tests use exec.Command instead of cmd.NewConnectCmd().Execute()
//     because Execute() with default flags starts foreground port-forwarding
//     that blocks indefinitely.
func ConnectSpec() {
	Describe("vCluster connect",
		labels.CLI,
		func() {
			var (
				vClusterName      string
				vClusterNamespace string
			)

			BeforeEach(func(ctx context.Context) context.Context {
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterNamespace = "vcluster-" + vClusterName
				return ctx
			})

			It("should print kubeconfig and use it to access the vCluster", func(ctx context.Context) {
				cmdCtx, cancel := context.WithTimeout(ctx, constants.PollingTimeout)
				defer cancel()

				By("running vcluster connect --print and capturing kubeconfig", func() {
					cmd := exec.CommandContext(cmdCtx, "vcluster", "connect",
						"-n", vClusterNamespace,
						"--print",
						vClusterName)
					kubeConfigBytes, err := cmd.Output()
					Expect(err).To(Succeed(),
						"vcluster connect --print failed for %s", vClusterName)
					Expect(kubeConfigBytes).NotTo(BeEmpty(), "printed kubeconfig should not be empty")

					restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeConfigBytes)
					Expect(err).To(Succeed(), "parsing printed kubeconfig")

					vClusterClient, err := kubernetes.NewForConfig(restConfig)
					Expect(err).To(Succeed(), "building kube client from printed kubeconfig")

					Eventually(func(g Gomega) {
						_, err := vClusterClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
						g.Expect(err).To(Succeed(),
							"listing pods via printed kubeconfig failed for %s", vClusterName)
					}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeoutShort).Should(Succeed())
				})
			})

			It("should fail to connect to a vCluster with an invalid name", func(ctx context.Context) {
				By("running vcluster connect --print with a non-existent vcluster name", func() {
					cmdCtx, cancel := context.WithTimeout(ctx, constants.PollingTimeoutShort)
					defer cancel()
					cmd := exec.CommandContext(cmdCtx, "vcluster", "connect",
						"-n", "INVALID",
						"--print",
						"INVALID")
					out, err := cmd.CombinedOutput()
					Expect(err).To(HaveOccurred(),
						"expected vcluster connect to fail for non-existent vcluster INVALID, output: %s", string(out))
				})
			})

			It("should connect to a vCluster and execute a command inline", func(ctx context.Context) {
				By("running vcluster connect with an inline kubectl command", func() {
					cmdCtx, cancel := context.WithTimeout(ctx, constants.PollingTimeout)
					defer cancel()
					cmd := exec.CommandContext(cmdCtx, "vcluster", "connect",
						"-n", vClusterNamespace,
						vClusterName,
						"--", "kubectl", "get", "ns")
					out, err := cmd.CombinedOutput()
					Expect(err).To(Succeed(),
						"vcluster connect -- kubectl get ns failed for %s, output: %s", vClusterName, string(out))
				})
			})
		},
	)
}
