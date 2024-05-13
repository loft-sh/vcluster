package reset

import (
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/loftctl/v4/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v4/pkg/upgrade"
	"github.com/spf13/cobra"
)

// NewResetCmd creates a new cobra command
func NewResetCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	description := product.ReplaceWithHeader("reset", "")
	if upgrade.IsPlugin == "true" {
		description = `
#######################################################
################### devspace reset ####################
#######################################################
	`
	}
	c := &cobra.Command{
		Use:   "reset",
		Short: "Reset configuration",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	c.AddCommand(NewPasswordCmd(globalFlags))
	return c
}
