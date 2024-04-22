package cmd

import (
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
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
		Long: `
#######################################################
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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Manager, "manager", "", "The manager to use for managing the virtual cluster, can be either helm or platform.")
	cobraCmd.Flags().StringVar(&cmd.Output, "output", "table", "Choose the format of the output. [table|json]")

	return cobraCmd
}

// Run executes the functionality
func (cmd *ListCmd) Run(cobraCmd *cobra.Command, _ []string) error {
	manager, err := platform.GetManager(cmd.Manager)
	if err != nil {
		return err
	}

	// check if we should create a platform vCluster
	if manager == platform.ManagerPlatform {
		return cli.ListPlatform(cobraCmd.Context(), &cmd.ListOptions, cmd.GlobalFlags, cmd.log)
	}

	return cli.ListHelm(cobraCmd.Context(), &cmd.ListOptions, cmd.GlobalFlags, cmd.log)
}
