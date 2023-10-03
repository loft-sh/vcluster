package list

import (
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/spf13/cobra"
)

// NewListCmd creates a new cobra command
func NewListCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	description := product.ReplaceWithHeader("list", "")
	if upgrade.IsPlugin == "true" {
		description = `
#######################################################
#################### devspace list ####################
#######################################################
	`
	}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Lists configuration",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	listCmd.AddCommand(NewTeamsCmd(globalFlags))
	listCmd.AddCommand(NewSpacesCmd(globalFlags))
	listCmd.AddCommand(NewClustersCmd(globalFlags))
	listCmd.AddCommand(NewVirtualClustersCmd(globalFlags))
	listCmd.AddCommand(NewSharedSecretsCmd(globalFlags))
	return listCmd
}
