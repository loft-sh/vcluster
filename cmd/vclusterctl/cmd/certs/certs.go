package certs

import (
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

func NewCertsCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	certsCmd := &cobra.Command{
		Use:   "certs",
		Short: "vCluster certs subcommands",
		Long: `#######################################################
################### vcluster certs ####################
#######################################################
	`,
		Args: cobra.NoArgs,
	}

	certsCmd.AddCommand(rotate(globalFlags))
	certsCmd.AddCommand(rotateCA(globalFlags))
	certsCmd.AddCommand(check(globalFlags))
	return certsCmd
}
