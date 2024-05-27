package get

import (
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

// NewGetCmd creates a new cobra command for the sub command
func NewGetCmd(globalFlags *flags.GlobalFlags, cfg *config.CLI) *cobra.Command {
	description := product.ReplaceWithHeader("var", "")

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Retrieves and display informations",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newClusterCmd(globalFlags, cfg))
	return cmd
}
