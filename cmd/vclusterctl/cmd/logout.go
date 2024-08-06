package cmd

import (
	"github.com/loft-sh/log"
	platformcli "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

func NewLogoutCmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	cmd := platformcli.NewLogoutCmd(globalFlags)
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
			log.GetInstance().Warnf("\"vcluster logout\" is deprecated, please use \"vcluster platform logout\" instead")
			return cmd.Run(cobraCmd.Context())
		},
	}

	return logoutCmd, nil
}
