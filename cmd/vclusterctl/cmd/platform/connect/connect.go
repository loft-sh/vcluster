package connect

import (
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/spf13/cobra"
)

// NewConnectCmd creates a new command
func NewConnectCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	connectCmd := &cobra.Command{
		Use:   "connect",
		Short: "Connects a cluster to vCluster.Pro",
		Long: `#######################################################
################ vcluster pro connect #################
#######################################################
		`,
		Args: cobra.NoArgs,
	}

	connectCmd.AddCommand(NewClusterCmd(globalFlags))
	return connectCmd
}
