package cmd

import (
	"fmt"

	loftctl "github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/procli"
	"github.com/spf13/cobra"
)

func NewLogoutCmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	loftctlGlobalFlags, err := procli.GlobalFlags(globalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pro flags: %w", err)
	}

	cmd := &loftctl.LogoutCmd{
		GlobalFlags: loftctlGlobalFlags,
		Log:         log.GetInstance(),
	}

	description := `########################################################
################### vcluster logout ####################
########################################################
Log out of vCluster.Pro

Example:
vcluster logout
########################################################
	`

	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out of a vCluster.Pro instance",
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			_, err := procli.CreateProClient()
			if err != nil {
				return err
			}

			return cmd.RunLogout(cobraCmd.Context(), args)
		},
	}

	return logoutCmd, nil
}
