package cmd

import (
	"cmp"
	"context"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/flags/create"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/spf13/cobra"
)

// CreateCmd holds the login cmd flags
type CreateCmd struct {
	*flags.GlobalFlags
	cli.CreateOptions

	log log.Logger
}

// NewCreateCmd creates a new command
func NewCreateCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &CreateCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "create" + util.VClusterNameOnlyUseLine,
		Short: "Create a new virtual cluster",
		Long: `#######################################################
################### vcluster create ###################
#######################################################
Creates a new virtual cluster

Example:
vcluster create test --namespace test
#######################################################
	`,
		Args: util.VClusterNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Manager, "manager", "", "The manager to use for managing the virtual cluster, can be either helm or platform.")

	create.AddCommonFlags(cobraCmd, &cmd.CreateOptions)
	create.AddHelmFlags(cobraCmd, &cmd.CreateOptions)
	create.AddPlatformFlags(cobraCmd, &cmd.CreateOptions, "[PLATFORM] ")

	return cobraCmd
}

// Run executes the functionality
func (cmd *CreateCmd) Run(ctx context.Context, args []string) error {
	cfg := cmd.LoadedConfig(cmd.log)

	// If manager has been passed as flag use it, otherwise read it from the config file
	managerType, err := config.ParseManagerType(cmp.Or(cmd.Manager, string(cfg.Manager.Type)))
	if err != nil {
		return fmt.Errorf("parse manager type: %w", err)
	}

	// check if there is a platform client or we skip the info message
	_, err = platform.InitClientFromConfig(ctx, cfg)
	if err == nil {
		config.PrintManagerInfo("create", cfg.Manager.Type, cmd.log)
	}
	// check if we should create a platform vCluster
	if managerType == config.ManagerPlatform {
		return cli.CreatePlatform(ctx, &cmd.CreateOptions, cmd.GlobalFlags, args[0], cmd.log)
	}

	return cli.CreateHelm(ctx, &cmd.CreateOptions, cmd.GlobalFlags, args[0], cmd.log)
}
