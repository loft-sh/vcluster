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

	proxy.AddCmd("create", "Create vcluster pro resources")
	proxy.AddCmd("create space", "Creates a new space in the given cluster")
	proxy.AddCmd("create vcluster", "Creates a new virtual cluster in the given parent cluster")

	proxy.AddCmd("delete", "Deletes vcluster pro resources")
	proxy.AddCmd("delete space", "Deletes a space from a cluster")
	proxy.AddCmd("delete vcluster", "Deletes a virtual cluster from a cluster")

	proxy.AddCmd("generate", "Generates configuration")
	proxy.AddCmd("generate admin-kube-config", "Generates a new kube config for connecting a cluster")

	proxy.AddCmd("get", "Get configuration")
	proxy.AddCmd("get secret", "Returns the key value of a project / shared secret")

	proxy.AddCmd("import", "Import vcluster pro resources")
	proxy.AddCmd("import space", "Imports a vcluster into a vcluster pro project")
	proxy.AddCmd("import vcluster", "Imports a space into a vcluster pro project")

	proxy.AddCmd("list", "List configuration")
	proxy.AddCmd("list spaces", "Lists the vcluster pro spaces you have access to")
	proxy.AddCmd("list vclusters", "Lists the vcluster pro virtual clusters you have access to")
	proxy.AddCmd("list secrets", "Lists all the shared secrets you have access to")

	proxy.AddCmd("reset", "Reset configuration")
	proxy.AddCmd("reset password", "Resets the password of a user")

	proxy.AddCmd("set", "Set configuration")
	proxy.AddCmd("set secret", "Sets the key value of a project / shared secret")

	proxy.AddCmd("use", "Uses vcluster pro resources")
	proxy.AddCmd("use space", "Creates a kube context for the given space")
	proxy.AddCmd("use vcluster", "Creates a kube context for the given virtual cluster")

	proCmd.AddCommand(proxy.Commands()...)

	return proCmd
}
