package pro

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/spf13/cobra"
)

func NewLoginCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to the vcluster.pro server",
		Long: `
#######################################################
#################### vcluster pro login #####################
#######################################################
Log in to the vcluster pro server
#######################################################
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.GetInstance().Info("Logging in to vcluster pro server ...")

			return nil
		},
	}

	return loginCmd
}
