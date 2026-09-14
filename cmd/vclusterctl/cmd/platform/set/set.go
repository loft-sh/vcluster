package set

import (
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/spf13/cobra"
)

// NewSetCmd creates a new cobra command
func NewSetCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	description := product.ReplaceWithHeader("set", "")
	c := &cobra.Command{
		Use:   "set",
		Short: "Set configuration",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	c.AddCommand(NewSecretCmd(globalFlags, defaults))
	return c
}
