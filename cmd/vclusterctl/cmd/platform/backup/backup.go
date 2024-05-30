package backup

import (
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

// NewAddCmd creates a new command
func NewBackupCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	addCmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup subcommands",
		Long: `#######################################################
############ vcluster platform backup #################
#######################################################
		`,
		Args: cobra.NoArgs,
	}

	addCmd.AddCommand(newManagementCmd(globalFlags))
	return addCmd
}
