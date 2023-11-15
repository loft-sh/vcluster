package vars

import (
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/spf13/cobra"
)

// NewVarsCmd creates a new cobra command for the sub command
func NewVarsCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	description := product.ReplaceWithHeader("var", "")

	if upgrade.IsPlugin == "true" {
		description = `
#######################################################
################### devspace vars #####################
#######################################################
`
	}

	cmd := &cobra.Command{
		Use:   "vars",
		Short: "Print predefined variables",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newUsernameCmd(globalFlags))
	cmd.AddCommand(newClusterCmd(globalFlags))
	return cmd
}
