package platform

import (
	"context"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/use"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

type LogoutCmd struct {
	*flags.GlobalFlags

	Log log.Logger
}

func NewLogoutCmd(gf *flags.GlobalFlags) LogoutCmd {
	return LogoutCmd{
		GlobalFlags: gf,
		Log:         log.GetInstance(),
	}
}

func NewLogoutCobraCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &LogoutCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := `########################################################
############## vcluster platform logout ################
########################################################
Log out of vCluster platform

Example:
vcluster platform logout
########################################################
	`

	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out of a vCluster platform instance",
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	return logoutCmd
}

func (cmd *LogoutCmd) Run(ctx context.Context) error {
	platformClient := platform.NewClientFromConfig(cmd.LoadedConfig(cmd.Log))

	// delete old access key if were logged in before
	cfg := platformClient.Config()
	if cfg.Platform.AccessKey != "" {
		if err := platformClient.Logout(ctx); err != nil {
			cmd.Log.Errorf("failed to send logout request: %v", err)
		}

		configHost := cfg.Platform.Host
		cfg.Platform.Host = ""
		cfg.Platform.AccessKey = ""
		cfg.Platform.LastInstallContext = ""
		cfg.Platform.Insecure = false

		if err := platformClient.Save(); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		cmd.Log.Donef("Successfully logged out of vCluster Palatform instance %s", ansi.Color(configHost, "white+b"))
	}

	return use.SwitchDriver(ctx, cfg, string(config.HelmDriver), cmd.Log)
}
