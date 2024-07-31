package etcd

import (
	"github.com/spf13/cobra"
)

func NewEtcdCmd() *cobra.Command {
	debugCmd := &cobra.Command{
		Use:   "etcd",
		Short: "vCluster etcd subcommand",
		Long: `#######################################################
############### vcluster debug etcd ###############
#######################################################
		`,
		Args: cobra.NoArgs,
	}

	debugCmd.AddCommand(NewKeysCommand())
	return debugCmd
}
