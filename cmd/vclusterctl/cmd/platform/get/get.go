package get

import (
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/spf13/cobra"
)

func NewGetCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults, cfg *config.CLI) *cobra.Command {
	description := product.ReplaceWithHeader("var", "")

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Retrieves and display informations",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newClusterCmd(globalFlags, cfg))
	cmd.AddCommand(NewSecretCmd(globalFlags, defaults))
	return cmd
}
