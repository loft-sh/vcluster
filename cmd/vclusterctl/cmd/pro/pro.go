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

	alias := NewAliasCmd(globalFlags)

	alias.AddCmd("create", "TODO: Fill out this description")
	alias.AddCmd("create space", "TODO: Fill out this description")
	alias.AddCmd("create vcluster", "TODO: Fill out this description")

	alias.AddCmd("delete", "TODO: Fill out this description")

	alias.AddCmd("generate", "TODO: Fill out this description")
	alias.AddCmd("generate admin-kube-config", "TODO: Fill out this description")

	alias.AddCmd("get", "TODO: Fill out this description")
	alias.AddCmd("get secret", "TODO: Fill out this description")

	alias.AddCmd("import", "TODO: Fill out this description")
	alias.AddCmd("list", "TODO: Fill out this description")

	alias.AddCmd("reset", "TODO: Fill out this description")
	alias.AddCmd("reset password", "TODO: Fill out this description")

	alias.AddCmd("set", "TODO: Fill out this description")
	alias.AddCmd("set secret", "TODO: Fill out this description")

	alias.AddCmd("use", "TODO: Fill out this description")
	alias.AddCmd("use space", "TODO: Fill out this description")
	alias.AddCmd("use vcluster", "TODO: Fill out this description")

	proCmd.AddCommand(alias.Commands()...)

	return proCmd
}
