// Package lifecycle contains vCluster CLI lifecycle tests (connect, pause/resume, etc.).
package lifecycle

import (
	"context"
	"os"
	"os/exec"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	loftlog "github.com/loft-sh/log"
	connectcmd "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// ConnectSpec registers all vcluster connect tests.
// These tests exercise the connect command against the suite-provided shared vCluster.
// Each It block is independent: it writes its kubeconfig to a fresh temp file and
// does not mutate shared cluster state.
func ConnectSpec() {
	Describe("vCluster connect",
		labels.PR,
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

			It("should connect to a vCluster and write kubeconfig to a file", func(ctx context.Context) {
				kcfgFile, err := os.CreateTemp("", "vcluster-connect-kubeconfig-*")
				Expect(err).To(Succeed(), "creating temp kubeconfig file")
				kcfgFile.Close()
				DeferCleanup(func(_ context.Context) { _ = os.Remove(kcfgFile.Name()) })

				By("running vcluster connect and writing kubeconfig to a temp file", func() {
					connectCmd := connectcmd.ConnectCmd{
						CobraCmd: &cobra.Command{},
						Log:      loftlog.Discard,
						GlobalFlags: &flags.GlobalFlags{
							Namespace: vClusterNamespace,
						},
						ConnectOptions: cli.ConnectOptions{
							KubeConfig:      kcfgFile.Name(),
							UpdateCurrent:   false,
							BackgroundProxy: false,
						},
					}
					Expect(connectCmd.Run(ctx, []string{vClusterName})).To(Succeed(),
						"vcluster connect failed for vcluster %s in namespace %s", vClusterName, vClusterNamespace)
				})

				By("verifying the kubeconfig file is non-empty", func() {
					data, err := os.ReadFile(kcfgFile.Name())
					Expect(err).To(Succeed())
					Expect(data).NotTo(BeEmpty(), "kubeconfig file should not be empty after connect")
				})
			})

			It("should print kubeconfig to stdout and use it to access the vCluster", func(ctx context.Context) {
				kcfgFile, err := os.CreateTemp("", "vcluster-print-kubeconfig-*")
				Expect(err).To(Succeed(), "creating temp kubeconfig file")
				DeferCleanup(func(_ context.Context) { _ = os.Remove(kcfgFile.Name()) })

				By("running vcluster connect --print via the CLI binary and capturing stdout", func() {
					// vcluster CLI must be in $PATH (same requirement as the old test).
					// --print writes kubeconfig YAML to stdout; capture it and write to file.
					connectCmd := exec.CommandContext(ctx, "vcluster", "connect", "--print",
						"-n", vClusterNamespace, vClusterName)
					kubeConfigBytes, err := connectCmd.Output()
					Expect(err).To(Succeed(),
						"vcluster connect --print failed for %s: %v", vClusterName, err)
					_, err = kcfgFile.Write(kubeConfigBytes)
					Expect(err).To(Succeed())
					Expect(kcfgFile.Close()).To(Succeed())
				})

				By("using the printed kubeconfig to list pods in the vCluster", func() {
					data, err := os.ReadFile(kcfgFile.Name())
					Expect(err).To(Succeed())
					Expect(data).NotTo(BeEmpty(), "printed kubeconfig should not be empty")

					restConfig, err := clientcmd.RESTConfigFromKubeConfig(data)
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
				By("running vcluster connect with a non-existent vcluster name", func() {
					// vcluster CLI must be in $PATH.
					connectCmd := exec.CommandContext(ctx, "vcluster", "connect", "--print",
						"-n", "INVALID", "INVALID")
					err := connectCmd.Run()
					Expect(err).To(HaveOccurred(),
						"expected vcluster connect to fail for non-existent vcluster INVALID")
					Expect(err.Error()).To(ContainSubstring("not found"),
						"expected 'not found' error for invalid vcluster, got: %v", err)
				})
			})

			It("should connect to a vCluster and execute a command inline", func(ctx context.Context) {
				By("running vcluster connect with an inline kubectl command", func() {
					connectCmd := connectcmd.ConnectCmd{
						CobraCmd: &cobra.Command{},
						Log:      loftlog.Discard,
						GlobalFlags: &flags.GlobalFlags{
							Namespace: vClusterNamespace,
						},
						ConnectOptions: cli.ConnectOptions{
							BackgroundProxy: false,
						},
					}
					Expect(connectCmd.Run(ctx, []string{vClusterName, "--", "kubectl", "get", "ns"})).To(Succeed(),
						"vcluster connect -- kubectl get ns failed for %s", vClusterName)
				})
			})

			It("should fail when connecting with an unreachable server override", func(ctx context.Context) {
				kcfgFile, err := os.CreateTemp("", "vcluster-unreachable-kubeconfig-*")
				Expect(err).To(Succeed(), "creating temp kubeconfig file")
				kcfgFile.Close()
				DeferCleanup(func(_ context.Context) { _ = os.Remove(kcfgFile.Name()) })

				By("running vcluster connect with an unreachable --server override", func() {
					connectCmd := connectcmd.ConnectCmd{
						CobraCmd: &cobra.Command{},
						Log:      loftlog.Discard,
						GlobalFlags: &flags.GlobalFlags{
							Namespace: vClusterNamespace,
						},
						ConnectOptions: cli.ConnectOptions{
							KubeConfig:      kcfgFile.Name(),
							Server:          "testdomain.org",
							UpdateCurrent:   false,
							BackgroundProxy: false,
						},
					}
					err := connectCmd.Run(ctx, []string{vClusterName})
					Expect(err).To(HaveOccurred(),
						"expected vcluster connect to fail with unreachable server testdomain.org")
					Expect(err.Error()).To(Or(
						ContainSubstring("timeout"),
						ContainSubstring("connection refused"),
						ContainSubstring("no such host")),
						"expected network error for unreachable server, got: %v", err)
				})
			})
		},
	)
}
