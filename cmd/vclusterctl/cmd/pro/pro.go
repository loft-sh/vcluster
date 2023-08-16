package pro

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/spf13/cobra"
)

func NewProCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	proCmd := &cobra.Command{
		Use:   "pro",
		Short: "vcluster.pro subcommands",
		Long: `
#######################################################
#################### vcluster get #####################
#######################################################
		`,
		Args: cobra.NoArgs,
	}

	proCmd.AddCommand(NewStartCmd(globalFlags))
	proCmd.AddCommand(NewLoginCmd(globalFlags))
	proCmd.AddCommand(NewCreateCmd(globalFlags))
	proCmd.AddCommand(NewImportCmd(globalFlags))
	proCmd.AddCommand(NewDeleteCmd(globalFlags))
	proCmd.AddCommand(NewListCmd(globalFlags))

	return proCmd
}
