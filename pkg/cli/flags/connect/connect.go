package connect

import (
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/spf13/cobra"
)

func AddCommonFlags(cmd *cobra.Command, options *cli.ConnectOptions) {
	cmd.Flags().StringVar(&options.KubeConfigContextName, "kube-config-context-name", "", "If set, will override the context name of the generated virtual cluster kube config with this name")
	cmd.Flags().StringVar(&options.KubeConfig, "kube-config", "./kubeconfig.yaml", "Writes the created kube config to this file")
	cmd.Flags().BoolVar(&options.UpdateCurrent, "update-current", true, "If true updates the current kube config")
	cmd.Flags().BoolVar(&options.Print, "print", false, "When enabled prints the context to stdout")
	cmd.Flags().StringVar(&options.PodName, "pod", "", "The pod to connect to")
	cmd.Flags().StringVar(&options.Server, "server", "", "The server to connect to")
	cmd.Flags().IntVar(&options.LocalPort, "local-port", 0, "The local port to forward the virtual cluster to. If empty, vCluster will use a random unused port")
	cmd.Flags().StringVar(&options.Address, "address", "", "The local address to start port forwarding under")
	cmd.Flags().StringVar(&options.ServiceAccount, "service-account", "", "If specified, vCluster will create a service account token to connect to the virtual cluster instead of using the default client cert / key. Service account must exist and can be used as namespace/name.")
	cmd.Flags().StringVar(&options.ServiceAccountClusterRole, "cluster-role", "", "If specified, vCluster will create the service account if it does not exist and also add a cluster role binding for the given cluster role to it. Requires --service-account to be set")
	cmd.Flags().IntVar(&options.ServiceAccountExpiration, "token-expiration", 0, "If specified, vCluster will create the service account token for the given duration in seconds. Defaults to eternal")
	cmd.Flags().BoolVar(&options.Insecure, "insecure", false, "If specified, vCluster will create the kube config with insecure-skip-tls-verify")
	cmd.Flags().BoolVar(&options.BackgroundProxy, "background-proxy", true, "Try to use a background-proxy to access the vCluster. Only works if docker is installed and reachable")

	// deprecated
	_ = cmd.Flags().MarkDeprecated("kube-config", fmt.Sprintf("please use %q to write the kubeconfig of the virtual cluster to stdout.", "vcluster connect --print"))
	_ = cmd.Flags().MarkDeprecated("kube-config-context-name", fmt.Sprintf("please use %q to write the kubeconfig of the virtual cluster to stdout.", "vcluster connect --print"))
	_ = cmd.Flags().MarkDeprecated("update-current", fmt.Sprintf("please use %q to write the kubeconfig of the virtual cluster to stdout.", "vcluster connect --print"))
}

func AddPlatformFlags(cmd *cobra.Command, options *cli.ConnectOptions, prefixes ...string) {
	prefix := strings.Join(prefixes, "")

	cmd.Flags().StringVar(&options.Project, "project", "", fmt.Sprintf("%sThe platform project the vCluster is in", prefix))
}
