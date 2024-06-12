package list

import (
	"github.com/loft-sh/api/v4/pkg/product"

	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

// NewListCmd creates a new cobra command
func NewListCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	description := product.ReplaceWithHeader("list", "")
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Lists configuration",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	listCmd.AddCommand(newClustersCmd(globalFlags))
	listCmd.AddCommand(newSharedSecretsCmd(globalFlags))
	listCmd.AddCommand(newTeamsCmd(globalFlags))
	listCmd.AddCommand(newVClustersCmd(globalFlags))
	listCmd.AddCommand(newNamespacesCmd(globalFlags))
	return listCmd
}
