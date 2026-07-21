package deletecmd

import (
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/spf13/cobra"
)

// NewDeleteCmd creates a new cobra command
func NewDeleteCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	description := product.ReplaceWithHeader("delete", "")
	c := &cobra.Command{
		Use:     "delete",
		Short:   product.Replace("Deletes vCluster platform resources"),
		Long:    description,
		Aliases: []string{"rm"},
		Args:    cobra.NoArgs,
	}
	c.AddCommand(newNamespaceCmd(globalFlags, defaults))
	c.AddCommand(newVClusterCmd(globalFlags))
	return c
}
