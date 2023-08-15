package pro

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/spf13/cobra"
)

func NewDeleteCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	deleteCmd := &cobra.Command{
		Use:   "delete [flags] vcluster_name",
		Short: "Delete a pro virtual cluster",
		Long: `
#######################################################
#################### vcluster pro delete #####################
#######################################################
Deletes a pro virtual cluster

Example:
vcluster pro delete test --namespace test
#######################################################
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.GetInstance().Info("Deleting pro virtual cluster ...")

			return nil
		},
	}

	return deleteCmd
}
