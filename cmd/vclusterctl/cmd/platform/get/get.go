package get

import (
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/spf13/cobra"
)

// NewGetCmd creates a new cobra command for the sub command
func NewGetCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults, cfg *config.CLI) *cobra.Command {
	description := product.ReplaceWithHeader("get", "")

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Retrieves and display informations",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newClusterCmd(globalFlags, cfg))
	cmd.AddCommand(newClusterAccessKeyCmd(globalFlags, cfg))
	cmd.AddCommand(newSecretCmd(globalFlags, defaults, cfg))
	cmd.AddCommand(newUserCmd(globalFlags, cfg))
	return cmd
}
