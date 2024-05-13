package wakeup

import (
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/loftctl/v4/cmd/loftctl/flags"
	pdefaults "github.com/loft-sh/loftctl/v4/pkg/defaults"
	"github.com/loft-sh/loftctl/v4/pkg/upgrade"
	"github.com/spf13/cobra"
)

// NewWakeUpCmd creates a new cobra command
func NewWakeUpCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	description := product.ReplaceWithHeader("wakeup", "")
	if upgrade.IsPlugin == "true" {
		description = `
#######################################################
################### devspace wakeup ###################
#######################################################
	`
	}
	cmd := &cobra.Command{
		Use:   "wakeup",
		Short: "Wakes up a space or vcluster",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(NewSpaceCmd(globalFlags, defaults))
	cmd.AddCommand(NewVClusterCmd(globalFlags, defaults))
	return cmd
}
