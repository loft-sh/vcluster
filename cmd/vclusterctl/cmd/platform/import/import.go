package importcmd

import (
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/spf13/cobra"
)

func NewImportCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	description := `
########################################################
############### vcluster platform import ###############
########################################################
`

	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Imports a virtual cluster / space into a vCluster platform project",
		Long:  description,
		Args:  util.VClusterNameOnlyValidator,
	}

	importCmd.AddCommand(NewVClusterCmd(globalFlags))
	importCmd.AddCommand(NewSpaceCmd(globalFlags))
	return importCmd
}
