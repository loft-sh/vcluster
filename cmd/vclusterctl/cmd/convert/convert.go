package convert

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/spf13/cobra"
)

func NewConvertCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	convertCmd := &cobra.Command{
		Use:   "convert",
		Short: "Convert virtual cluster config values",
		Long: `
#######################################################
################## vcluster convert ###################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	convertCmd.AddCommand(convertValues(globalFlags))
	return convertCmd
}
