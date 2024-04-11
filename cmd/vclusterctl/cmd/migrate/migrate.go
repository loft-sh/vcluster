package migrate

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/spf13/cobra"
)

func NewMigrateCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate vcluster configuration",
		Long: `
#######################################################
################## vcluster migrate ###################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	migrateCmd.AddCommand(migrateValues(globalFlags))
	return migrateCmd
}
