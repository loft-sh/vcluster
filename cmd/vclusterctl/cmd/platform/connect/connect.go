package connect

import (
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	platformdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/spf13/cobra"
)

// NewConnectCmd creates a new cobra command
func NewConnectCmd(globalFlags *flags.GlobalFlags, _ *platformdefaults.Defaults) *cobra.Command {
	description := product.ReplaceWithHeader("use", `

Activates a kube context for the given cluster / space / vcluster / management.
	`)
	useCmd := &cobra.Command{
		Use:   "connect",
		Short: product.Replace("Uses loft resources"),
		Long:  description,
		Args:  cobra.NoArgs,
	}

	useCmd.AddCommand(NewClusterCmd(globalFlags))
	return useCmd
}
