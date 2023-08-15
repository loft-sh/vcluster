package pro

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/spf13/cobra"
)

func NewStartCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Starts the vcluster.pro server",
		Long: `
#######################################################
#################### vcluster pro start #####################
#######################################################
Starts the vcluster pro server
#######################################################
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.GetInstance().Info("Starting vcluster pro server ...")

			return nil
		},
	}

	return startCmd
}
