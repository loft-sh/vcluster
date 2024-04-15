package cmd

import (
	"fmt"
	"os"

	"github.com/loft-sh/api/v3/pkg/product"
	loftctl "github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/procli"
	"github.com/spf13/cobra"
)

func NewLoginCmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	loftctlGlobalFlags, err := procli.GlobalFlags(globalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pro flags: %w", err)
	}

	cmd := &loftctl.LoginCmd{
		GlobalFlags: loftctlGlobalFlags,
		Log:         log.GetInstance(),
	}

	description := `########################################################
#################### vcluster login ####################
########################################################
Login into vCluster.Pro

Example:
vcluster login https://my-vcluster-pro.com
vcluster login https://my-vcluster-pro.com --access-key myaccesskey
########################################################
	`

	loginCmd := &cobra.Command{
		Use:   "login [VCLUSTER_PRO_HOST]",
		Short: "Login to a vCluster.Pro instance",
		Long:  description,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			if config.ShouldCheckForProFeatures() {
				cmd.Log.Warnf("In order to use a Pro feature, please contact us at https://www.vcluster.com/pro-demo or downgrade by running `vcluster upgrade --version v0.19.5`")
				os.Exit(0)
			}

			return cmd.RunLogin(cobraCmd.Context(), args)
		},
	}

	loginCmd.Flags().StringVar(&cmd.AccessKey, "access-key", "", "The access key to use")
	loginCmd.Flags().BoolVar(&cmd.Insecure, "insecure", true, product.Replace("Allow login into an insecure Loft instance"))
	loginCmd.Flags().BoolVar(&cmd.DockerLogin, "docker-login", true, "If true, will log into the docker image registries the user has image pull secrets for")

	return loginCmd, nil
}
