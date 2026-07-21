package connect

import (
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/spf13/cobra"
)

// NewConnectCmd creates a new cobra command
func NewConnectCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults) *cobra.Command {
	description := product.ReplaceWithHeader("connect", `

Activates a kube context for the given cluster / management / namespace / vcluster.
	`)
	connectCmd := &cobra.Command{
		Use:   "connect",
		Short: product.Replace("connects to vCluster platform resources"),
		Long:  description,
		Args:  cobra.NoArgs,
	}

	connectCmd.AddCommand(newClusterCmd(globalFlags))
	connectCmd.AddCommand(newManagementCmd(globalFlags))
	connectCmd.AddCommand(newNamespaceCmd(globalFlags, defaults))
	connectCmd.AddCommand(newVClusterCmd(globalFlags))
	return connectCmd
}
