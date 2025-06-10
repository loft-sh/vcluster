package cmd

import (
	"cmp"
	"context"
	"fmt"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/spf13/cobra"
)

// PauseCmd holds the cmd flags
type PauseCmd struct {
	*flags.GlobalFlags
	cli.PauseOptions

	Log log.Logger
}

// NewPauseCmd creates a new command
func NewPauseCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &PauseCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:     "pause" + util.VClusterNameOnlyUseLine,
		Aliases: []string{"sleep"},
		Short:   "Pauses a virtual cluster",
		Long: `#######################################################
################### vcluster pause ####################
#######################################################
Pause will stop a virtual cluster and free all its used
computing resources.

Pause will scale down the virtual cluster and delete
all workloads created through the virtual cluster. Upon resume,
all workloads will be recreated. Other resources such
as persistent volume claims, services etc. will not be affected.

Example:
vcluster pause test --namespace test
#######################################################
	`,
		Args:              util.VClusterNameOnlyValidator,
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Driver, "driver", "", "The driver for the virtual cluster, can be either helm or platform.")

	// Platform flags
	cobraCmd.Flags().StringVar(&cmd.Project, "project", "", "[PLATFORM] The vCluster platform project to use")
	cobraCmd.Flags().Int64Var(&cmd.ForceDuration, "prevent-wakeup", -1, product.Replace("[PLATFORM] The amount of seconds this vcluster should sleep until it can be woken up again (use 0 for infinite sleeping). During this time the space can only be woken up by `vcluster resume vcluster`, manually deleting the annotation on the namespace or through the loft UI"))

	return cobraCmd
}

// Run executes the functionality
func (cmd *PauseCmd) Run(ctx context.Context, args []string) error {
	cfg := cmd.LoadedConfig(cmd.Log)

	// If driver has been passed as flag use it, otherwise read it from the config file
	driverType, err := config.ParseDriverType(cmp.Or(cmd.Driver, string(cfg.Driver.Type)))
	if err != nil {
		return fmt.Errorf("parse driver type: %w", err)
	}

	// check if we should create a platform vCluster
	if driverType == config.PlatformDriver {
		return cli.PausePlatform(ctx, &cmd.PauseOptions, cfg, args[0], cmd.Log)
	}

	return cli.PauseHelm(ctx, cmd.GlobalFlags, args[0], cmd.Log)
}
