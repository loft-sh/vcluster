package cmd

import (
	"fmt"

	loftctl "github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd"
	loftctlflags "github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/spf13/cobra"
)

func NewLogoutCmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	loftctlGlobalFlags := &loftctlflags.GlobalFlags{
		Silent:    globalFlags.Silent,
		Debug:     globalFlags.Debug,
		LogOutput: globalFlags.LogOutput,
	}

	if globalFlags.Config != "" {
		loftctlGlobalFlags.Config = globalFlags.Config
	} else {
		var err error
		loftctlGlobalFlags.Config, err = pro.GetLoftConfigFilePath()
		if err != nil {
			return nil, fmt.Errorf("failed to get vcluster pro configuration file path: %w", err)
		}
	}

	logoutCmd := loftctl.NewLogoutCmd(loftctlGlobalFlags)

	logoutCmd.Use = "logout"
	logoutCmd.Long = `########################################################
Log out of vCluster.Pro

Example:
vcluster logout
########################################################
	`

	return logoutCmd, nil
}
