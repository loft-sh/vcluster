package wakeup

import (
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/spf13/cobra"
)

// NewWakeupCmd creates a new cobra command
func NewWakeupCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults) *cobra.Command {
	description := product.ReplaceWithHeader("wakeup", `

Wake up a virtual cluster / namespace.
	`)
	wakeupCmd := &cobra.Command{
		Use:   "wakeup",
		Short: "Wake up a virtual cluster / namespace",
		Long:  description,
		Args:  cobra.NoArgs,
	}

	wakeupCmd.AddCommand(NewVClusterCmd(globalFlags))
	wakeupCmd.AddCommand(NewNamespaceCmd(globalFlags, defaults))
	return wakeupCmd
}
