package create

import (
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/spf13/cobra"
)

// NewCreateCmd creates a new cobra command
func NewCreateCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	description := product.ReplaceWithHeader("create", "")
	c := &cobra.Command{
		Use:   "create",
		Short: product.Replace("Creates vCluster platform resources"),
		Long:  description,
		Args:  cobra.NoArgs,
	}
	c.AddCommand(newNamespaceCmd(globalFlags, defaults))
	c.AddCommand(newVClusterCmd(globalFlags))
	return c
}
