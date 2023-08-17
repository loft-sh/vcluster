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

	AddAliasCmd(globalFlags, "list")
	AddAliasCmd(globalFlags, "import")
	AddAliasCmd(globalFlags, "delete")

	AddAliasCmd(globalFlags, "get secret")
	AddAliasCmd(globalFlags, "set secret")

	AddAliasCmd(globalFlags, "use space")
	AddAliasCmd(globalFlags, "use vcluster")
	AddAliasCmd(globalFlags, "create space")
	AddAliasCmd(globalFlags, "create vcluster")
	AddAliasCmd(globalFlags, "generate admin-kube-config")
	AddAliasCmd(globalFlags, "reset password")

	proCmd.AddCommand(GetRootCmds()...)

	return proCmd
}
