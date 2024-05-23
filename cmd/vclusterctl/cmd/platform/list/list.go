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

	// TODO: change that with the actual globalFlag variable
	listCmd.AddCommand(NewClustersCmd(globalFlags))
	listCmd.AddCommand(NewSharedSecretsCmd(globalFlags))
	return listCmd
}
