package node

import (
	"context"

	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/spf13/cobra"
)

type UpgradeOptions struct {
	pro.UpgradeOptions
}

func NewUpgradeCommand() *cobra.Command {
	options := &UpgradeOptions{}

	upgradeCmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade the node",

		RunE: func(cmd *cobra.Command, _ []string) error {
			return Run(cmd.Context(), &options.UpgradeOptions)
		},
	}

	upgradeCmd.Flags().StringVar(&options.KubernetesVersion, "kubernetes-version", "", "The Kubernetes version to upgrade to")
	upgradeCmd.Flags().StringVar(&options.BundleRepository, "bundle-repository", "https://github.com/loft-sh/kubernetes/releases/download", "The repository to use for downloading the Kubernetes bundle")
	upgradeCmd.Flags().StringVar(&options.BinariesPath, "binaries-path", "/usr/local/bin", "The path to the kubeadm binaries")
	upgradeCmd.Flags().StringVar(&options.CNIBinariesPath, "cni-binaries-path", "/opt/cni/bin", "The path to the CNI binaries")

	return upgradeCmd
}

func Run(ctx context.Context, options *pro.UpgradeOptions) error {
	return pro.UpgradeNode(ctx, options)
}
