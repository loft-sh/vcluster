package set

import (
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	pdefaults "github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/spf13/cobra"
)

// NewSetCmd creates a new cobra command
func NewSetCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	description := product.ReplaceWithHeader("set", "")
	if upgrade.IsPlugin == "true" {
		description = `
#######################################################
#################### devspace set #####################
#######################################################
	`
	}
	c := &cobra.Command{
		Use:   "set",
		Short: "Set configuration",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	c.AddCommand(NewSecretCmd(globalFlags, defaults))
	return c
}
