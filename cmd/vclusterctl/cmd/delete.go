package cmd

import (
	"cmp"
	"context"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	flagsdelete "github.com/loft-sh/vcluster/pkg/cli/flags/delete"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/spf13/cobra"
)

// DeleteCmd holds the delete cmd flags
type DeleteCmd struct {
	*flags.GlobalFlags
	cli.DeleteOptions

	log log.Logger
}

// NewDeleteCmd creates a new command
func NewDeleteCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &DeleteCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "delete" + util.VClusterNameOnlyUseLine,
		Short: "Deletes a virtual cluster",
		Long: `#######################################################
################### vcluster delete ###################
#######################################################
Deletes a virtual cluster

Example:
vcluster delete test --namespace test
#######################################################
	`,
		Args:              util.VClusterNameOnlyValidator,
		Aliases:           []string{"rm"},
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Manager, "manager", "", "The manager to use for managing the virtual cluster, can be either helm or platform.")

	flagsdelete.AddCommonFlags(cobraCmd, &cmd.DeleteOptions)
	flagsdelete.AddHelmFlags(cobraCmd, &cmd.DeleteOptions)
	flagsdelete.AddPlatformFlags(cobraCmd, &cmd.DeleteOptions, "[PLATFORM] ")

	return cobraCmd
}

// Run executes the functionality
func (cmd *DeleteCmd) Run(ctx context.Context, args []string) error {
	cfg := cmd.LoadedConfig(cmd.log)

	// If manager has been passed as flag use it, otherwise read it from the config file
	managerType, err := config.ParseManagerType(cmp.Or(cmd.Manager, string(cfg.Manager.Type)))
	if err != nil {
		return fmt.Errorf("parse manager type: %w", err)
	}
	// check if there is a platform client or we skip the info message
	_, err = platform.InitClientFromConfig(ctx, cfg)
	if err == nil {
		config.PrintManagerInfo("delete", cfg.Manager.Type, cmd.log)
	}

	if managerType == config.ManagerPlatform {
		return cli.DeletePlatform(ctx, &cmd.DeleteOptions, cfg, args[0], cmd.log)
	}

	return cli.DeleteHelm(ctx, &cmd.DeleteOptions, cmd.GlobalFlags, args[0], cmd.log)
}
