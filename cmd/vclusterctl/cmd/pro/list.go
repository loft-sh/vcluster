package pro

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/spf13/cobra"
)

func NewListCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all pro virtual clusters",
		Long: `
#######################################################
#################### vcluster pro list #####################
#######################################################
List all pro virtual cluster

Example:
vcluster list
vcluster list --output json
vcluster list --namespace test
#######################################################
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.GetInstance().Info("Listing pro virtual clusters ...")

			return nil
		},
	}

	return listCmd
}
