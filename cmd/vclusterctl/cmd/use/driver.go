package use

import (
	"context"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/spf13/cobra"
)

type DriverCmd struct {
	*flags.GlobalFlags

	Log log.Logger
}

func NewDriverCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &DriverCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := `########################################################
################# vcluster use driver #################
########################################################
Either use "helm" or "platform" as the deployment method for managing virtual clusters.
#######################################################
	`

	driverCmd := &cobra.Command{
		Use:   "driver",
		Short: "Switch the virtual clusters driver between platform and helm",
		Long:  description,
		Args:  cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	return driverCmd
}

func (cmd *DriverCmd) Run(ctx context.Context, args []string) error {
	return SwitchDriver(ctx, cmd.LoadedConfig(cmd.Log), args[0], cmd.Log)
}

func SwitchDriver(ctx context.Context, cfg *config.CLI, driver string, log log.Logger) error {
	driverType, err := config.ParseDriverType(driver)
	if err != nil {
		return fmt.Errorf("parse driver type: %w", err)
	}

	if driverType == config.PlatformDriver {
		_, err := platform.InitClientFromConfig(ctx, cfg)
		if err != nil {
			return fmt.Errorf("cannot switch to platform driver because it seems like you are not logged into a vCluster platform (%w)", err)
		}
	}

	cfg.Driver.Type = driverType
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save vCluster config: %w", err)
	}

	log.Donef("Successfully switched driver to %s", driver)

	return nil
}
