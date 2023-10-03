package sleep

import (
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	pdefaults "github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/spf13/cobra"
)

// NewSleepCmd creates a new cobra command
func NewSleepCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	description := product.ReplaceWithHeader("sleep", "")
	if upgrade.IsPlugin == "true" {
		description = `
#######################################################
################### devspace sleep ####################
#######################################################
	`
	}
	cmd := &cobra.Command{
		Use:   "sleep",
		Short: "Puts spaces or vclusters to sleep",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(NewSpaceCmd(globalFlags, defaults))
	cmd.AddCommand(NewVClusterCmd(globalFlags, defaults))
	return cmd
}
