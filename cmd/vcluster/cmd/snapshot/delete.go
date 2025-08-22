package snapshot

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/spf13/cobra"
)

func NewDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "delete vCluster snapshot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			options := &Options{}
			envOptions, err := snapshot.ParseOptionsFromEnv()
			if err != nil {
				return fmt.Errorf("failed to parse options from environment: %w", err)
			}
			options.Snapshot = *envOptions

			return options.Delete(cmd.Context())
		},
	}

	return cmd
}
