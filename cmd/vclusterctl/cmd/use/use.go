package use

import (
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

func NewUseCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	useCmd := &cobra.Command{
		Use:   "use",
		Short: "vCluster use subcommand",
		Long: `#######################################################
#################### vcluster use #####################
#######################################################
		`,
		Args: cobra.NoArgs,
	}

	useCmd.AddCommand(NewDriverCmd(globalFlags))
	return useCmd
}
