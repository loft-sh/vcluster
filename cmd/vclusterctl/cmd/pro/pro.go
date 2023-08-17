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
#################### vcluster pro #####################
#######################################################
		`,
		Args: cobra.NoArgs,
	}

	proCmd.AddCommand(NewStartCmd(globalFlags))
	proCmd.AddCommand(NewLoginCmd(globalFlags))

	proCmd.AddCommand(NewAliasCmd(globalFlags, "list"))
	proCmd.AddCommand(NewAliasCmd(globalFlags, "create"))
	proCmd.AddCommand(NewAliasCmd(globalFlags, "import"))
	proCmd.AddCommand(NewAliasCmd(globalFlags, "delete"))
	proCmd.AddCommand(NewAliasCmd(globalFlags, "use"))
	proCmd.AddCommand(NewAliasCmd(globalFlags, "generate"))
	proCmd.AddCommand(NewAliasCmd(globalFlags, "reset"))

	return proCmd
}
