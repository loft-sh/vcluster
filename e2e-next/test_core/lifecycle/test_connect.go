// Package lifecycle contains vCluster CLI lifecycle tests (connect, pause/resume, etc.).
package lifecycle

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// ConnectSpec registers vcluster connect tests.
// All tests shell out to the vcluster binary (must be in $PATH).
//
// Migrated from test/e2e_cli/connect/connect.go. Changes from old suite:
//   - Old "connect by name" and "connect --print to file" merged into one test
//     because both validate the same thing: connect produces a usable kubeconfig.
//   - Old "unreachable server" test removed: --print with --server just embeds
//     the server in the kubeconfig without connecting, so the error case is untestable.
//   - All tests use exec.Command with --background-proxy=false to prevent the CLI
//     from starting a Docker proxy container.
//   - --print test passes --server pointing to the framework's existing background
//     proxy so the CLI exits immediately instead of starting its own port-forward.

// vclusterBin returns the path to the vcluster binary.
// Uses $GOBIN/vcluster if GOBIN is set (same as e2e-framework provider),
// falls back to "vcluster" which relies on $PATH.
func vclusterBin() string {
	if gobin := os.Getenv("GOBIN"); gobin != "" {
		return filepath.Join(gobin, "vcluster")
	}
	return "vcluster"
}

func ConnectSpec() {
	Describe("vCluster connect", func() {
		var (
			vClusterName      string
			vClusterNamespace string
			proxyServer       string
		)

		BeforeEach(func(ctx context.Context) context.Context {
			vClusterName = cluster.CurrentClusterNameFrom(ctx)
			vClusterNamespace = "vcluster-" + vClusterName
			// Get the server address from the framework's background proxy.
			proxyServer = cluster.CurrentClusterFrom(ctx).KubernetesRestConfig().Host
			return ctx
		})

		It("should print kubeconfig and use it to access the vCluster", func(ctx context.Context) {
			cmdCtx, cancel := context.WithTimeout(ctx, constants.PollingTimeout)
			defer cancel()

			By("running vcluster connect --print and capturing kubeconfig", func() {
				cmd := exec.CommandContext(cmdCtx, vclusterBin(), "connect",
					"-n", vClusterNamespace,
					"--print",
					"--background-proxy=false",
					"--server", proxyServer,
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
				cmd := exec.CommandContext(cmdCtx, vclusterBin(), "connect",
					"-n", "INVALID",
					"--print",
					"--background-proxy=false",
					"INVALID")
				out, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred(),
					"expected vcluster connect to fail for non-existent vcluster INVALID, output: %s", string(out))
				Expect(string(out)).To(ContainSubstring("find"),
					"expected error output to mention finding vcluster, got: %s", string(out))
			})
		})

		It("should connect to a vCluster and execute a command inline", func(ctx context.Context) {
			By("running vcluster connect with an inline kubectl command", func() {
				// Retry the whole command because port-forwarding can fail
				// transiently in CI (e.g. containerd closing the pod network
				// namespace mid-forward).
				Eventually(func(g Gomega) {
					cmdCtx, cancel := context.WithTimeout(ctx, constants.PollingTimeoutShort)
					defer cancel()
					cmd := exec.CommandContext(cmdCtx, vclusterBin(), "connect",
						"-n", vClusterNamespace,
						"--background-proxy=false",
						vClusterName,
						"--", "kubectl", "get", "ns")
					out, err := cmd.CombinedOutput()
					g.Expect(err).To(Succeed(),
						"vcluster connect -- kubectl get ns failed for %s, output: %s", vClusterName, string(out))
				}).WithPolling(constants.PollingInterval).WithTimeout(constants.PollingTimeout).Should(Succeed())
			})
		})
	},
	)
}
