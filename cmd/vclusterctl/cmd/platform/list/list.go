package list

import (
	"github.com/loft-sh/api/v4/pkg/product"

	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

// NewListCmd creates a new cobra command
func NewListCmd(globalFlags *flags.GlobalFlags, cfg *config.CLI) *cobra.Command {
	description := product.ReplaceWithHeader("list", "")
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Lists configuration",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	listCmd.AddCommand(newClustersCmd(globalFlags, cfg))
	listCmd.AddCommand(newSharedSecretsCmd(globalFlags, cfg))
	listCmd.AddCommand(newTeamsCmd(globalFlags, cfg))
	return listCmd
}
