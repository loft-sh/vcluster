package cmd

import (
	"context"
	"fmt"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/use"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
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
	platformClient, err := platform.NewClientFromConfig(ctx, cmd.LoadedConfig(cmd.Log))
	if err != nil {
		return err
	}
	cfg := platformClient.Config()

	// delete old access key if were logged in before
	if cfg.Platform.AccessKey != "" {
		if err := platformClient.Logout(ctx); err != nil {
			return err
		}
		configHost := cfg.Platform.Host

		cfg.Platform.Host = ""
		cfg.Platform.AccessKey = ""
		cfg.Platform.LastInstallContext = ""
		cfg.Platform.Insecure = false

		if err := platformClient.Save(); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		cmd.Log.Donef(product.Replace("Successfully logged out of loft instance %s"), ansi.Color(configHost, "white+b"))
	}

	return use.SwitchManager(ctx, cfg, string(config.ManagerHelm), cmd.Log)
}
