package registry

import (
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

func NewRegistryCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	registryCmd := &cobra.Command{
		Use:   "registry",
		Short: "vCluster registry subcommand",
		Long: `#######################################################
################### vcluster registry ####################
#######################################################
		`,
		Args: cobra.NoArgs,
	}

	registryCmd.AddCommand(NewPushCmd(globalFlags))
	registryCmd.AddCommand(NewPullCmd(globalFlags))
	registryCmd.AddCommand(NewProxyCmd(globalFlags))
	return registryCmd
}
