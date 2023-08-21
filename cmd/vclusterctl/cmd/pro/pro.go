//go:build pro
// +build pro

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

	proxy := NewAliasCmd(globalFlags)

	proxy.AddCmd("create", "TODO: Fill out this description")
	proxy.AddCmd("create space", "TODO: Fill out this description")
	proxy.AddCmd("create vcluster", "TODO: Fill out this description")

	proxy.AddCmd("delete", "TODO: Fill out this description")

	proxy.AddCmd("generate", "TODO: Fill out this description")
	proxy.AddCmd("generate admin-kube-config", "TODO: Fill out this description")

	proxy.AddCmd("get", "TODO: Fill out this description")
	proxy.AddCmd("get secret", "TODO: Fill out this description")

	proxy.AddCmd("import", "TODO: Fill out this description")
	proxy.AddCmd("list", "TODO: Fill out this description")

	proxy.AddCmd("reset", "TODO: Fill out this description")
	proxy.AddCmd("reset password", "TODO: Fill out this description")

	proxy.AddCmd("set", "TODO: Fill out this description")
	proxy.AddCmd("set secret", "TODO: Fill out this description")

	proxy.AddCmd("use", "TODO: Fill out this description")
	proxy.AddCmd("use space", "TODO: Fill out this description")
	proxy.AddCmd("use vcluster", "TODO: Fill out this description")

	proCmd.AddCommand(proxy.Commands()...)

	return proCmd
}
