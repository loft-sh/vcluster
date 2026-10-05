package mappings

import (
	"github.com/spf13/cobra"
)

func NewMappingsCmd() *cobra.Command {
	debugCmd := &cobra.Command{
		Use:   "mappings",
		Short: "vCluster mappings subcommand",
		Long: `#######################################################
############### vcluster debug mappings ###############
#######################################################
		`,
		Args: cobra.NoArgs,
	}

	debugCmd.AddCommand(NewListCommand())
	debugCmd.AddCommand(NewClearCommand())
	debugCmd.AddCommand(NewAddCommand())
	debugCmd.AddCommand(NewDeleteCommand())
	return debugCmd
}
