package standalone

import (
	"github.com/loft-sh/vcluster/pkg/cli/standalone"
	"github.com/spf13/cobra"
)

func NewResetCommand() *cobra.Command {
	var config string

	resetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Performs a best effort revert of changes made to this host by 'vcluster standalone install'",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return standalone.Reset(cmd.Context(), config)
		},
		Aliases: []string{"uninstall"},
	}

	resetCmd.Flags().StringVar(&config, "config", "", "Path to the vcluster.yaml configuration file")

	return resetCmd
}
