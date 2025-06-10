package get

import (
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/spf13/cobra"
)

// NewGetCmd creates a new cobra command for the sub command
func NewGetCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults) *cobra.Command {
	description := product.ReplaceWithHeader("get", "")

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Retrieves and display informations",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newClusterCmd(globalFlags))
	cmd.AddCommand(newClusterAccessKeyCmd(globalFlags))
	cmd.AddCommand(newSecretCmd(globalFlags, defaults))
	cmd.AddCommand(newUserCmd(globalFlags))
	return cmd
}
