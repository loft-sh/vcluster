package get

import (
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	pdefaults "github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/spf13/cobra"
)

// NewGetCmd creates a new cobra command
func NewGetCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	description := product.ReplaceWithHeader("get", "")
	if upgrade.IsPlugin == "true" {
		description = `
#######################################################
#################### devspace get #####################
#######################################################
`
	}
	c := &cobra.Command{
		Use:   "get",
		Short: "Get configuration",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	c.AddCommand(NewUserCmd(globalFlags))
	c.AddCommand(NewSecretCmd(globalFlags, defaults))
	c.AddCommand(NewClusterAccessKeyCmd(globalFlags))
	return c
}
