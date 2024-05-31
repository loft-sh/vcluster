package sleep

import (
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

// NewSleepCmd creates a new cobra command
func NewSleepCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	description := product.ReplaceWithHeader("sleep", `

Put a virtual cluster to sleep.
	`)
	wakeupCmd := &cobra.Command{
		Use:   "sleep",
		Short: product.Replace("Put a virtual cluster to sleep"),
		Long:  description,
		Args:  cobra.NoArgs,
	}

	wakeupCmd.AddCommand(NewVClusterCmd(globalFlags))
	return wakeupCmd
}
