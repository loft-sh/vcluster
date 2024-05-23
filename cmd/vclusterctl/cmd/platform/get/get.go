package get

import (
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/spf13/cobra"
)

// NewVarsCmd creates a new cobra command for the sub command
func NewVarsCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	description := product.ReplaceWithHeader("var", "")

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Retrieves and display informations",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newClusterCmd(globalFlags))
	cmd.AddCommand(NewSecretCmd(globalFlags, defaults))
	return cmd
}
