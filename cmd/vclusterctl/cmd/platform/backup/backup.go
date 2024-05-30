package backup

import (
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

// NewAddCmd creates a new command
func NewBackupCmd(globalFlags *flags.GlobalFlags, cfg *config.CLI) *cobra.Command {
	addCmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup subcommands",
		Long: `#######################################################
############ vcluster platform backup #################
#######################################################
		`,
		Args: cobra.NoArgs,
	}

	addCmd.AddCommand(newManagementCmd(globalFlags, cfg))
	return addCmd
}
