package get

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/spf13/cobra"
)

func NewGetCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Gets cluster related information",
		Long: `
#######################################################
#################### vcluster get #####################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	getCmd.AddCommand(getServiceCIDR(globalFlags))
	return getCmd
}
