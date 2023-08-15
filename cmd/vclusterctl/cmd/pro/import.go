package pro

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/spf13/cobra"
)

func NewImportCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import an existing pro virtual cluster to the vcluster pro server",
		Long: `
#######################################################
#################### vcluster pro import #####################
#######################################################
Import an existing pro virtual cluster to the vcluster pro server
#######################################################
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.GetInstance().Info("Importing pro virtual cluster ...")

			return nil
		},
	}

	return importCmd
}
