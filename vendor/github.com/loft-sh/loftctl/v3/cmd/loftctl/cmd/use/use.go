package use

import (
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	pdefaults "github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/spf13/cobra"
)

// NewUseCmd creates a new cobra command
func NewUseCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	description := product.ReplaceWithHeader("use", `

Activates a kube context for the given cluster / space / vcluster / management.
	`)
	if upgrade.IsPlugin == "true" {
		description = `
#######################################################
#################### devspace use #####################
#######################################################

Activates a kube context for the given cluster / space / vcluster / management.
	`
	}
	useCmd := &cobra.Command{
		Use:   "use",
		Short: product.Replace("Uses loft resources"),
		Long:  description,
		Args:  cobra.NoArgs,
	}

	useCmd.AddCommand(NewClusterCmd(globalFlags))
	useCmd.AddCommand(NewManagementCmd(globalFlags))
	useCmd.AddCommand(NewSpaceCmd(globalFlags, defaults))
	useCmd.AddCommand(NewVirtualClusterCmd(globalFlags, defaults))
	return useCmd
}
