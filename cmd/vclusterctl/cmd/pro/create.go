package pro

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/spf13/cobra"
)

func NewCreateCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new pro virtual cluster",
		Long: `
#######################################################
#################### vcluster pro create #####################
#######################################################
Creates a new pro virtual cluster

Example:
vcluster pro create test --namespace test
#######################################################
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.GetInstance().Info("Creating a new vcluster pro...")

			return nil
		},
	}

	return createCmd
}
