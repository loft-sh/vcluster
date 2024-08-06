package cmd

import (
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	platformcli "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/spf13/cobra"
)

const PlatformURL = "VCLUSTER_PLATFORM_URL"

type LoginCmd struct {
	*flags.GlobalFlags

	Log log.Logger

	Driver      string
	AccessKey   string
	Insecure    bool
	DockerLogin bool
}

func NewLoginCmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	cmd := platformcli.NewLoginCmd(globalFlags)

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
			log.GetInstance().Warnf("\"vcluster login\" is deprecated, please use \"vcluster platform login\" instead")
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	loginCmd.Flags().StringVar(&cmd.Driver, "use-driver", "", "Switch vCluster driver between platform and helm")
	loginCmd.Flags().StringVar(&cmd.AccessKey, "access-key", "", "The access key to use")
	loginCmd.Flags().BoolVar(&cmd.Insecure, "insecure", true, product.Replace("Allow login into an insecure Loft instance"))
	loginCmd.Flags().BoolVar(&cmd.DockerLogin, "docker-login", true, "If true, will log into the docker image registries the user has image pull secrets for")

	return loginCmd, nil
}
