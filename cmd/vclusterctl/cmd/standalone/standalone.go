package standalone

import "github.com/spf13/cobra"

func NewStandaloneCmd() *cobra.Command {
	standaloneCmd := &cobra.Command{
		Use:   "standalone",
		Short: "vCluster standalone subcommand",
		Long: `#######################################################
################ vcluster standalone ##################
#######################################################
		`,
		Args: cobra.NoArgs,
	}

	standaloneCmd.AddCommand(NewInstallCommand())
	standaloneCmd.AddCommand(NewResetCommand())
	return standaloneCmd
}
