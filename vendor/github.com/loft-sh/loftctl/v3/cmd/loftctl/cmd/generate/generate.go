package generate

import (
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/spf13/cobra"
)

// NewGenerateCmd creates a new cobra command
func NewGenerateCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	description := product.ReplaceWithHeader("generate", "")
	if upgrade.IsPlugin == "true" {
		description = `
#######################################################
################## devspace generate ##################
#######################################################
	`
	}
	c := &cobra.Command{
		Use:   "generate",
		Short: "Generates configuration",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	c.AddCommand(NewAdminKubeConfigCmd(globalFlags))
	return c
}
