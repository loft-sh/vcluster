package cmd

import (
	"cmp"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

// ListCmd holds the login cmd flags
type ListCmd struct {
	*flags.GlobalFlags
	cli.ListOptions

	log log.Logger
}

// NewListCmd creates a new command
func NewListCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ListCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "list",
		Short: "Lists all virtual clusters",
		Long: `#######################################################
#################### vcluster list ####################
#######################################################
Lists all virtual clusters

Example:
vcluster list
vcluster list --output json
vcluster list --namespace test
#######################################################
	`,
		Args:    cobra.NoArgs,
		Aliases: []string{"ls"},
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Manager, "manager", "", "The manager to use for managing the virtual cluster, can be either helm or platform.")
	cobraCmd.Flags().StringVar(&cmd.Output, "output", "table", "Choose the format of the output. [table|json]")

	return cobraCmd
}

// Run executes the functionality
func (cmd *ListCmd) Run(cobraCmd *cobra.Command) error {
	cfg := cmd.LoadedConfig(cmd.log)

	// If manager has been passed as flag use it, otherwise read it from the config file
	managerType, err := config.ParseManagerType(cmp.Or(cmd.Manager, string(cfg.Manager.Type)))
	if err != nil {
		return fmt.Errorf("parse manager type: %w", err)
	}
	if managerType == config.ManagerPlatform {
		return cli.ListPlatform(cobraCmd.Context(), &cmd.ListOptions, cmd.GlobalFlags, cmd.log)
	}

	return cli.ListHelm(cobraCmd.Context(), &cmd.ListOptions, cmd.GlobalFlags, cmd.log)
}
