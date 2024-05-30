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

type ManagerCmd struct {
	*flags.GlobalFlags

	Log log.Logger
}

func NewManagerCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ManagerCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := `########################################################
################# vcluster use manager #################
########################################################
Either use helm or vCluster platform as the deployment method for managing virtual clusters.
#######################################################
	`

	managerCmd := &cobra.Command{
		Use:   "manager",
		Short: "Switch managing method of virtual clusters between platform and helm",
		Long:  description,
		Args:  cobra.ExactArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			if args[0] != string(config.ManagerHelm) && args[0] != string(config.ManagerPlatform) {
				return fmt.Errorf("you can only use helm or platform to use")
			}

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	return managerCmd
}

func (cmd *ManagerCmd) Run(ctx context.Context, args []string) error {
	return SwitchManager(ctx, cmd.LoadedConfig(cmd.Log), args[0], cmd.Log)
}

func SwitchManager(ctx context.Context, cfg *config.CLI, manager string, log log.Logger) error {
	if cfg.Manager.Type == config.ManagerPlatform {
		_, err := platform.InitClientFromConfig(ctx, cfg)
		if err != nil {
			return fmt.Errorf("cannot switch to platform manager, because seems like you are not logged into a vCluster platform (%w)", err)
		}
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save vCluster config: %w", err)
	}

	log.Donef("Successfully switched manager to %s", manager)

	return nil
}
