package share

import (
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/loftctl/v4/cmd/loftctl/flags"
	pdefaults "github.com/loft-sh/loftctl/v4/pkg/defaults"
	"github.com/loft-sh/loftctl/v4/pkg/upgrade"
	"github.com/spf13/cobra"
)

// NewShareCmd creates a new cobra command
func NewShareCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	description := product.ReplaceWithHeader("share", "")
	if upgrade.IsPlugin == "true" {
		description = `
#######################################################
################### devspace share ####################
#######################################################
	`
	}
	cmd := &cobra.Command{
		Use:   "share",
		Short: "Shares cluster resources",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(NewSpaceCmd(globalFlags, defaults))
	cmd.AddCommand(NewVClusterCmd(globalFlags, defaults))
	return cmd
}
