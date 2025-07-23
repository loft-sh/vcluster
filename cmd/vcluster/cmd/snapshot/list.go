package snapshot

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list vCluster snapshots",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			options := &SnapshotOptions{}
			envOptions, err := parseOptionsFromEnv()
			if err != nil {
				klog.Warningf("Error parsing environment variables: %v", err)
			} else {
				options.Snapshot = *envOptions
			}

			snapshots, err := options.List(cmd.Context())
			if err != nil {
				return fmt.Errorf("list snapshots: %w", err)
			}

			encodedBytes, err := json.Marshal(snapshots)
			if err != nil {
				return fmt.Errorf("json marshal: %w", err)
			}

			if _, err := os.Stdout.Write(encodedBytes); err != nil {
				return err
			}
			return nil
		},
	}

	return cmd
}
