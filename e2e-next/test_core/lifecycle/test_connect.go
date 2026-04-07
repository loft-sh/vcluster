// Package lifecycle contains vCluster CLI lifecycle tests (connect, pause/resume, etc.).
package lifecycle

import (
	"context"
	"os"
	"os/exec"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	connectcmd "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// ConnectSpec registers vcluster connect tests.
// Uses cmd.NewConnectCmd (cobra) for tests that need --/flag parsing,
// and exec.Command for tests that validate CLI binary behavior.
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

			It("should connect to a vCluster and write kubeconfig to a file", func(_ context.Context) {
				kcfgFile, err := os.CreateTemp("", "vcluster-connect-kubeconfig-*")
				Expect(err).To(Succeed(), "creating temp kubeconfig file")
				kcfgFile.Close()
				DeferCleanup(func(_ context.Context) { _ = os.Remove(kcfgFile.Name()) })

				By("running vcluster connect and writing kubeconfig to a temp file", func() {
					cmd := connectcmd.NewConnectCmd(&flags.GlobalFlags{
						Namespace: vClusterNamespace,
					})
					Expect(cmd.Flags().Set("kube-config", kcfgFile.Name())).To(Succeed())
					cmd.SetArgs([]string{vClusterName})
					Expect(cmd.Execute()).To(Succeed(),
						"vcluster connect failed for %s in %s", vClusterName, vClusterNamespace)
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

				By("running vcluster connect --print and capturing stdout", func() {
					cmdCtx, cancel := context.WithTimeout(ctx, constants.PollingTimeout)
					defer cancel()
					cmd := exec.CommandContext(cmdCtx, "vcluster", "connect", "-n", vClusterNamespace, "--print", vClusterName)
					kubeConfigBytes, err := cmd.Output()
					Expect(err).To(Succeed(),
						"vcluster connect --print failed for %s", vClusterName)
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
				By("running vcluster connect --print with a non-existent vcluster name", func() {
					cmdCtx, cancel := context.WithTimeout(ctx, constants.PollingTimeoutShort)
					defer cancel()
					cmd := exec.CommandContext(cmdCtx, "vcluster", "connect", "-n", "INVALID", "--print", "INVALID")
					out, err := cmd.CombinedOutput()
					Expect(err).To(HaveOccurred(),
						"expected vcluster connect to fail for non-existent vcluster INVALID, output: %s", string(out))
				})
			})

			It("should connect to a vCluster and execute a command inline", func(_ context.Context) {
				By("running vcluster connect with an inline kubectl command", func() {
					cmd := connectcmd.NewConnectCmd(&flags.GlobalFlags{
						Namespace: vClusterNamespace,
					})
					cmd.SetArgs([]string{vClusterName, "--", "kubectl", "get", "ns"})
					Expect(cmd.Execute()).To(Succeed(),
						"vcluster connect -- kubectl get ns failed for %s", vClusterName)
				})
			})

			It("should fail when connecting with an unreachable server override", func(_ context.Context) {
				kcfgFile, err := os.CreateTemp("", "vcluster-unreachable-kubeconfig-*")
				Expect(err).To(Succeed(), "creating temp kubeconfig file")
				kcfgFile.Close()
				DeferCleanup(func(_ context.Context) { _ = os.Remove(kcfgFile.Name()) })

				By("running vcluster connect with an unreachable --server override", func() {
					cmd := connectcmd.NewConnectCmd(&flags.GlobalFlags{
						Namespace: vClusterNamespace,
					})
					Expect(cmd.Flags().Set("kube-config", kcfgFile.Name())).To(Succeed())
					Expect(cmd.Flags().Set("server", "testdomain.org")).To(Succeed())
					cmd.SetArgs([]string{vClusterName})
					err := cmd.Execute()
					Expect(err).To(HaveOccurred(),
						"expected vcluster connect to fail with unreachable server testdomain.org")
				})
			})
		},
	)
}
