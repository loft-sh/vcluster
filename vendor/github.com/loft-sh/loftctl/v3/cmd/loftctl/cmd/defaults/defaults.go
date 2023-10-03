package defaults

import (
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	pdefaults "github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/spf13/cobra"
)

// NewDefaultsCmd creates a new command
func NewDefaultsCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	description := product.ReplaceWithHeader("defaults", "")

	defaultsCmd := &cobra.Command{
		Use:   "defaults",
		Short: "Sets default values for loftctl",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	defaultsCmd.AddCommand(NewSetCmd(globalFlags, defaults))
	defaultsCmd.AddCommand(NewGetCmd(globalFlags, defaults))
	defaultsCmd.AddCommand(NewViewCmd(globalFlags, defaults))
	return defaultsCmd
}
