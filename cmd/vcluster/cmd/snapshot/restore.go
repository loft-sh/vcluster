package snapshot

import (
	"fmt"
	"os"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/spf13/cobra"
)

var (
	newVCluster    bool
	restoreVolumes bool
)

func NewRestoreCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore a vCluster",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			vConfig, err := config.LoadRuntimeConfig(os.Getenv("VCLUSTER_NAME"))
			if err != nil {
				return err
			}

			envOptions, err := snapshot.ParseOptionsFromEnv()
			if err != nil {
				return fmt.Errorf("failed to parse options from environment: %w", err)
			}
			restoreClient := snapshot.NewRestoreClient(*envOptions, restoreVolumes, newVCluster)
			return restoreClient.Run(cmd.Context(), vConfig)
		},
	}

	cmd.Flags().BoolVar(&newVCluster, "new-vcluster", false, "Restore a new vCluster from snapshot instead of restoring into an existing vCluster")
	cmd.Flags().BoolVar(&restoreVolumes, "restore-volumes", false, "Restore volumes from volume snapshots")
	return cmd
}
