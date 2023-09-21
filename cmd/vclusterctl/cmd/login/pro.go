//go:build pro
// +build pro

package login

import (
	"fmt"

	loftctl "github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd"
	loftctlflags "github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/spf13/cobra"
)

func NewLoginCmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	loftctlGlobalFlags := &loftctlflags.GlobalFlags{
		Silent:    globalFlags.Silent,
		Debug:     globalFlags.Debug,
		Config:    globalFlags.Config,
		LogOutput: globalFlags.LogOutput,
	}

	var err error
	loftctlGlobalFlags.Config, err = pro.GetConfigFilePath()
	if err != nil {
		return nil, fmt.Errorf("failed to get vcluster pro configuration file path: %w", err)
	}

	loginCmd := loftctl.NewLoginCmd(loftctlGlobalFlags)

	loginCmd.Use = "login [VCLUSTER_PRO_HOST]"
	loginCmd.Long = `########################################################
Login into vCluster.Pro

Example:
vcluster login https://my-vcluster-pro.com
vcluster login https://my-vcluster-pro.com --access-key myaccesskey
########################################################
	`

	return loginCmd, nil
}
