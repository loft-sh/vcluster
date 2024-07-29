package debug

import (
	"github.com/loft-sh/vcluster/cmd/vcluster/cmd/debug/etcd"
	"github.com/loft-sh/vcluster/cmd/vcluster/cmd/debug/mappings"
	"github.com/spf13/cobra"
)

func NewDebugCmd() *cobra.Command {
	debugCmd := &cobra.Command{
		Use:   "debug",
		Short: "vCluster debug subcommand",
		Long: `#######################################################
################### vcluster debug ####################
#######################################################
		`,
		Args: cobra.NoArgs,
	}

	debugCmd.AddCommand(mappings.NewMappingsCmd())
	debugCmd.AddCommand(etcd.NewEtcdCmd())
	return debugCmd
}
