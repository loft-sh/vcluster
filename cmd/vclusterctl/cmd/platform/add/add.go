package add

import (
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

// NewAddCmd creates a new command
func NewAddCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Adds a cluster to vCluster platform",
		Long: `#######################################################
############# vcluster platform add ###################
#######################################################
		`,
		Args: cobra.NoArgs,
	}

	addCmd.AddCommand(NewClusterCmd(globalFlags))
	addCmd.AddCommand(NewVClusterCmd(globalFlags))
	return addCmd
}
