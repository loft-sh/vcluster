package snapshot

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list vCluster snapshots",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			options := &Options{}
			envOptions, err := parseOptionsFromEnv()
			if err != nil {
				return fmt.Errorf("failed to parse options from environment: %w", err)
			}
			options.Snapshot = *envOptions

			snapshots, err := options.List(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to list snapshots: %w", err)
			}

			encodedBytes, err := json.Marshal(snapshots)
			if err != nil {
				return fmt.Errorf("failed to marshal json: %w", err)
			}

			if _, err := os.Stdout.Write(encodedBytes); err != nil {
				return err
			}
			return nil
		},
	}

	return cmd
}
