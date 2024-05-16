package cmd

import (
	"context"
	"fmt"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/use"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/manager"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

type LogoutCmd struct {
	*flags.GlobalFlags

	Log log.Logger
}

func NewLogoutCmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	cmd := &LogoutCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := `########################################################
################### vcluster logout ####################
########################################################
Log out of vCluster platform

Example:
vcluster logout
########################################################
	`

	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out of a vCluster platform instance",
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context())
		},
	}

	return logoutCmd, nil
}

func (cmd *LogoutCmd) Run(ctx context.Context) error {
	cfg := config.Read(cmd.Config, cmd.Log)
	platformClient, err := platform.CreateClientFromConfig(ctx, cfg.Platform.Config)
	if err != nil {
		return err
	}

	// delete old access key if were logged in before
	if platformClient.Config().AccessKey != "" {
		previousHost, err := platformClient.Logout(ctx)
		if err != nil {
			return fmt.Errorf("logout: %w", err)
		}

		platformConfig := platformClient.Config()
		cfg.Platform.Config = platformConfig

		if err := config.Write(cmd.Config, cfg); err != nil {
			return fmt.Errorf("save vCluster config: %w", err)
		}

		cmd.Log.Donef(product.Replace("Successfully logged out of loft instance %s"), ansi.Color(previousHost, "white+b"))
	}

	return use.SwitchManager(ctx, globalFlags.Config, string(manager.Helm), cmd.Log)
}
