package pro

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/spf13/cobra"
)

func NewStopCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stops the vcluster.pro server",
		Long: `
#######################################################
#################### vcluster pro stop #####################
#######################################################
Stops the vcluster pro server
#######################################################
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.GetInstance().Info("Stopping vcluster pro server ...")

			return nil
		},
	}

	return stopCmd
}
