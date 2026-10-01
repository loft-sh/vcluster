package certs

import (
	"github.com/spf13/cobra"
)

func NewCertsCmd() *cobra.Command {
	certsCmd := &cobra.Command{
		Use:   "certs",
		Short: "vCluster certs subcommands",
		Long: `#######################################################
################## vcluster certs #####################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	certsCmd.AddCommand(rotate())
	certsCmd.AddCommand(rotateCA())
	certsCmd.AddCommand(check())
	return certsCmd
}
