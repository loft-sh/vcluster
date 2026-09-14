package share

import (
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/spf13/cobra"
)

// NewShareCmd creates a new command
func NewShareCmd(globalFlags *flags.GlobalFlags, defaults *defaults.Defaults) *cobra.Command {
	shareCmd := &cobra.Command{
		Use:   "share",
		Short: "Shares a virtual cluster / namespace with another vCluster platform user or team",
		Long: `#########################################################
################ vcluster platform share ################
#########################################################
		`,
		Args:    cobra.NoArgs,
		Aliases: []string{"ls"},
	}

	shareCmd.AddCommand(NewVClusterCmd(globalFlags))
	shareCmd.AddCommand(NewNamespaceCmd(globalFlags, defaults))
	return shareCmd
}
