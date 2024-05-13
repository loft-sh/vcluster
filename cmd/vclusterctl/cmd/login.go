package cmd

import (
	"fmt"

	"github.com/loft-sh/api/v4/pkg/product"
	loftctl "github.com/loft-sh/loftctl/v4/cmd/loftctl/cmd"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/use"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/spf13/cobra"
)

type LoginOptions struct {
	Manager string

	AccessKey   string
	Insecure    bool
	DockerLogin bool
}

func NewLoginCmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	loftGlobalFlags, err := platform.GlobalFlags(globalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pro flags: %w", err)
	}

	options := &LoginOptions{}
	description := `########################################################
#################### vcluster login ####################
########################################################
Login into vCluster platform

Example:
vcluster login https://my-vcluster-platform.com
vcluster login https://my-vcluster-platform.com --access-key myaccesskey
########################################################
	`

	loginCmd := &cobra.Command{
		Use:   "login [VCLUSTER_PLATFORM_HOST]",
		Short: "Login to a vCluster platform instance",
		Long:  description,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			loginCmd := &loftctl.LoginCmd{
				GlobalFlags: loftGlobalFlags,

				AccessKey:   options.AccessKey,
				Insecure:    options.Insecure,
				DockerLogin: options.DockerLogin,

				Log: log.GetInstance(),
			}

			err = loginCmd.RunLogin(cobraCmd.Context(), args)
			if err != nil {
				return err
			}

			// should switch manager
			if options.Manager != "" {
				err = use.SwitchManager(options.Manager, log.GetInstance())
				if err != nil {
					return fmt.Errorf("switch manager failed: %w", err)
				}
			}

			return nil
		},
	}

	loginCmd.Flags().StringVar(&options.Manager, "use-manager", "", "Switch managing method of vClusters between platform and helm")

	loginCmd.Flags().StringVar(&options.AccessKey, "access-key", "", "The access key to use")
	loginCmd.Flags().BoolVar(&options.Insecure, "insecure", true, product.Replace("Allow login into an insecure Loft instance"))
	loginCmd.Flags().BoolVar(&options.DockerLogin, "docker-login", true, "If true, will log into the docker image registries the user has image pull secrets for")

	return loginCmd, nil
}
