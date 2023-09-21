package delete

import (
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/spf13/cobra"
)

// NewDeleteCmd creates a new cobra command
func NewDeleteCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults) *cobra.Command {
	description := product.ReplaceWithHeader("delete", "")
	if upgrade.IsPlugin == "true" {
		description = `
#######################################################
##################### loft delete #####################
#######################################################
	`
	}
	c := &cobra.Command{
		Use:   "delete",
		Short: product.Replace("Deletes loft resources"),
		Long:  description,
		Args:  cobra.NoArgs,
	}

	c.AddCommand(NewSpaceCmd(globalFlags, defaults))
	c.AddCommand(NewVirtualClusterCmd(globalFlags, defaults))
	return c
}
