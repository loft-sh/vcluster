package cmd

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/telemetry"

	"github.com/spf13/cobra"
)

func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the vCluster version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), telemetry.SyncerVersion)
		},
	}
}
