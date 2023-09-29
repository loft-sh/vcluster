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
		Short: "vCluster.Pro subcommands",
		Long: `#######################################################
#################### vcluster pro #####################
#######################################################
		`,
		Args: cobra.NoArgs,
	}

	proCmd.AddCommand(NewStartCmd(globalFlags))
	proCmd.AddCommand(NewLoginCmd(globalFlags))

	proxy := NewAliasCmd(globalFlags)

	proxy.AddCmd("connect", "Creates a kube context for the given virtual cluster")
	proxy.AddCmd("create", "Creates a new virtual cluster in the given parent cluster")
	proxy.AddCmd("delete", "Deletes a virtual cluster from a cluster")
	proxy.AddCmd("import", "Imports a vCluster.Pro instance into a space")
	proxy.AddCmd("list", "Lists the vCluster.Pro instances you have access to")
	// Use is an alias to connect
	proxy.AddCmd("use", "Creates a kube context for the given virtual cluster")

	proxy.AddCmd("secret", "Management Operations on secret resources")
	proxy.AddCmd("secret get", "Returns the key value of a project / shared secret")
	proxy.AddCmd("secret list", "Lists all the shared secrets you have access to")
	proxy.AddCmd("secret set", "Sets the key value of a project / shared secret")

	proxy.AddCmd("generate", "Generates configuration")
	proxy.AddCmd("generate admin-kube-config", "Generates a new kube config for connecting a cluster")

	proxy.AddCmd("reset", "Reset configuration")
	proxy.AddCmd("reset password", "Resets the password of a user")

	proCmd.AddCommand(proxy.Commands()...)

	return proCmd
}
