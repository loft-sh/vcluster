package cmd

import (
	loftctl "github.com/loft-sh/loftctl/v4/cmd/loftctl/cmd"
	loftctlflags "github.com/loft-sh/loftctl/v4/cmd/loftctl/flags"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/use"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/spf13/cobra"
)

func NewLogoutCmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	cmd := &loftctl.LogoutCmd{
		GlobalFlags: &loftctlflags.GlobalFlags{
			Config:    globalFlags.Config,
			LogOutput: globalFlags.LogOutput,
			Silent:    globalFlags.Silent,
			Debug:     globalFlags.Debug,
		},
		Log: log.GetInstance(),
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
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			_, err := platform.CreatePlatformClient()
			if err != nil {
				return err
			}

			err = cmd.RunLogout(cobraCmd.Context(), args)
			if err != nil {
				return err
			}

			err = use.SwitchManager(string(platform.ManagerHelm), log.GetInstance())
			if err != nil {
				return err
			}

			return err
		},
	}

	return logoutCmd, nil
}
