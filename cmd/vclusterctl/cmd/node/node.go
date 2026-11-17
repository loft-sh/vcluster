package node

import (
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

func NewNodeCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	nodeCmd := &cobra.Command{
		Use:   "node",
		Short: "vCluster node subcommand",
		Long: `#######################################################
################### vcluster debug ####################
#######################################################
		`,
		Args: cobra.NoArgs,
	}

	nodeCmd.AddCommand(NewUpgradeCommand(globalFlags))
	nodeCmd.AddCommand(NewLoadImageCommand(globalFlags))
	nodeCmd.AddCommand(NewDeleteCommand(globalFlags))
	return nodeCmd
}
