package snapshot

import (
	"fmt"
	"os"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/spf13/cobra"
)

func NewSnapshotCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Manage vCluster snapshots",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			vConfig, err := config.LoadConfig(os.Getenv("VCLUSTER_NAME"))
			if err != nil {
				return err
			}
			client := &snapshot.Client{}
			envOptions, err := snapshot.ParseOptionsFromEnv()
			if err != nil {
				return fmt.Errorf("failed to parse options from environment: %w", err)
			}
			client.Options = *envOptions

			return client.Run(cmd.Context(), vConfig)
		},
	}

	cmd.AddCommand(NewCreateCmd())
	cmd.AddCommand(NewGetCmd())
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewDeleteCmd())

	return cmd
}
