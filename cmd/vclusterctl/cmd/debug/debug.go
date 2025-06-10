package debug

import (
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

func NewDebugCommand(globalFlags *flags.GlobalFlags) *cobra.Command {
	convertCmd := &cobra.Command{
		Use:   "debug",
		Short: "Debug retrieves information from vCluster",
		Long: `#######################################################
################### vcluster debug ####################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	convertCmd.AddCommand(NewCollectCmd(globalFlags))
	return convertCmd
}
