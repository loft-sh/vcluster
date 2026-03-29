package test_cli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"time"

	"github.com/loft-sh/e2e-framework/pkg/setup/cluster"
	"github.com/loft-sh/e2e-framework/pkg/setup/suite"
	loftlog "github.com/loft-sh/log"
	connectcmd "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"
	"github.com/loft-sh/vcluster/e2e-next/clusters"
	"github.com/loft-sh/vcluster/e2e-next/constants"
	"github.com/loft-sh/vcluster/e2e-next/labels"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/util/kubeclient"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

// DescribeCLIConnect registers CLI connect/disconnect/list tests against the given vCluster.
func DescribeCLIConnect(vcluster suite.Dependency) bool {
	return Describe("vcluster CLI connect",
		Ordered,
		labels.PR,
		labels.CLI,
		cluster.Use(vcluster),
		cluster.Use(clusters.HostCluster),
		func() {
			var (
				vClusterName string
				vClusterNS   string
			)

			BeforeEach(func(ctx context.Context) {
				vClusterName = cluster.CurrentClusterNameFrom(ctx)
				vClusterNS = "vcluster-" + vClusterName
			})

			It("should connect to a vcluster and write kubeconfig to a temp file", func(ctx context.Context) {
				By("Creating a temp kubeconfig file", func() {
					tmpFile, err := os.CreateTemp("", "kubeconfig-connect-*.yaml")
					Expect(err).NotTo(HaveOccurred())
					Expect(tmpFile.Close()).To(Succeed())
					DeferCleanup(os.Remove, tmpFile.Name())

					connectCmd := connectcmd.ConnectCmd{
						CobraCmd: &cobra.Command{},
						Log:      loftlog.Discard,
						GlobalFlags: &flags.GlobalFlags{
							Namespace: vClusterNS,
						},
						ConnectOptions: cli.ConnectOptions{
							KubeConfig:           tmpFile.Name(),
							BackgroundProxy:      true,
							BackgroundProxyImage: constants.GetVClusterImage(),
						},
					}

					Expect(connectCmd.Run(ctx, []string{vClusterName})).To(Succeed(), "vcluster connect failed")
					out, readErr := os.ReadFile(tmpFile.Name())
					Expect(readErr).NotTo(HaveOccurred())

					By("Verifying the kubeconfig is valid YAML with a non-empty server ", func() {
						cfg, parseErr := clientcmd.Load(out)
						Expect(parseErr).To(Succeed(), "kubeconfig is not valid YAML")
						Expect(cfg.Clusters).NotTo(BeEmpty(), "kubeconfig has no clusters: %s", string(out))

						for _, clusterCfg := range cfg.Clusters {
							Expect(clusterCfg.Server).NotTo(BeEmpty(), "cluster server should not be empty")
						}
					})
				})
			})

			It("should fail to print kubeconfig for a non-existent vcluster", func(ctx context.Context) {
				By("Running vcluster connect --print for an invalid name", func() {
					connectExecCmd := exec.CommandContext(ctx, "vcluster", "connect", "-n", "INVALID-NS", "--print", "INVALID-NAME")
					err := connectExecCmd.Run()
					Expect(err).To(HaveOccurred(), "expected connect to fail for non-existent vcluster")
				})
			})

			It("should connect to a vcluster and execute a command", func(ctx context.Context) {
				By("Running vcluster connect with a trailing command", func() {
					withTrailingCmd := connectcmd.NewConnectCmd(&flags.GlobalFlags{})
					withTrailingCmd.SetArgs([]string{vClusterName, "--", "kubectl", "get", "ns"})

					err := withTrailingCmd.Execute()
					By("Verifying the command was executed ")
					Expect(err).NotTo(HaveOccurred(), "vcluster connect -- kubectl get ns failed")
				})
			})

			It("should fail when connecting with an invalid server override", func(ctx context.Context) {
				By("Attempting to connect with an unreachable server", func() {
					tmpFile, err := os.CreateTemp("", "kubeconfig-badserver-*.yaml")
					Expect(err).NotTo(HaveOccurred())
					Expect(tmpFile.Close()).To(Succeed())
					DeferCleanup(os.Remove, tmpFile.Name())

					connectCmd := connectcmd.ConnectCmd{
						CobraCmd: &cobra.Command{},
						Log:      loftlog.Discard,
						GlobalFlags: &flags.GlobalFlags{
							Namespace: vClusterNS,
						},
						ConnectOptions: cli.ConnectOptions{
							KubeConfig:           tmpFile.Name(),
							Server:               "https://testdomain.invalid:6443",
							BackgroundProxy:      true,
							BackgroundProxyImage: constants.GetVClusterImage(),
						},
					}

					runCtx, cancel := context.WithTimeout(ctx, time.Second*10)
					defer cancel()
					err = connectCmd.Run(runCtx, []string{vClusterName})
					Expect(err).To(HaveOccurred(), "expected connect to fail with invalid server")
				})
			})

			It("should list the vcluster in vcluster list output", func(ctx context.Context) {
				By("Running vcluster list via exec.Command", func() {
					listCmd := exec.CommandContext(ctx, "vcluster", "list", "-n", vClusterNS)
					var out bytes.Buffer
					listCmd.Stdout = &out
					listCmd.Stderr = &out

					err := listCmd.Run()
					Expect(err).To(Succeed(), "vcluster list failed: %s", out.String())
					Expect(out.String()).To(ContainSubstring(vClusterName),
						"expected vcluster name %q in list output", vClusterName)
				})
			})

			It("should connect then disconnect and restore the original context", func(ctx context.Context) {
				By("Recording the original kubeconfig context", func() {
					rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
						clientcmd.NewDefaultClientConfigLoadingRules(),
						&clientcmd.ConfigOverrides{},
					).RawConfig()
					Expect(err).NotTo(HaveOccurred())
					originalContext := rawConfig.CurrentContext

					By("Connecting to the vcluster and switching context", func() {
						connectCmd := connectcmd.ConnectCmd{
							CobraCmd: &cobra.Command{},
							Log:      loftlog.Discard,
							GlobalFlags: &flags.GlobalFlags{
								Namespace: vClusterNS,
							},
							ConnectOptions: cli.ConnectOptions{
								UpdateCurrent:        true,
								BackgroundProxy:      true,
								BackgroundProxyImage: constants.GetVClusterImage(),
							},
						}

						err = connectCmd.Run(ctx, []string{vClusterName})
						Expect(err).To(Succeed(), "vcluster connect --update-current failed")
					})

					By("Verifying the active context changed to the vcluster context", func() {
						expectedCtx := kubeclient.ContextName(vClusterName, vClusterNS, originalContext)
						currentCtx, _, err := kubeclient.CurrentContext()
						Expect(err).NotTo(HaveOccurred())
						Expect(currentCtx).To(Equal(expectedCtx),
							"expected active context to be %q after connect, got %q", expectedCtx, currentCtx)
					})

					By("Disconnecting and restoring the original context", func() {
						disconnectCmd := connectcmd.NewDisconnectCmd(&flags.GlobalFlags{})
						err = disconnectCmd.RunE(disconnectCmd, []string{})
						Expect(err).To(Succeed(), "vcluster disconnect failed")

						currentCtx, _, err := kubeclient.CurrentContext()
						Expect(err).NotTo(HaveOccurred())
						Expect(currentCtx).To(Equal(originalContext),
							"expected context %q after disconnect, got %q", originalContext, currentCtx)
					})
				})
			})
		},
	)
}
