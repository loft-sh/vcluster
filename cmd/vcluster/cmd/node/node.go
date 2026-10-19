package node

import (
	"github.com/spf13/cobra"
)

func NewNodeCmd() *cobra.Command {
	nodeCmd := &cobra.Command{
		Use:   "node",
		Short: "vCluster node subcommand",
		Long: `#######################################################
################### vcluster node ####################
#######################################################
		`,
		Args: cobra.NoArgs,
	}

	nodeCmd.AddCommand(NewUpgradeCommand())
	return nodeCmd
}
