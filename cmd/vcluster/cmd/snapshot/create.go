package snapshot

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	setupconfig "github.com/loft-sh/vcluster/pkg/setup/config"
	"k8s.io/client-go/kubernetes"

	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/spf13/cobra"
)

func NewCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create vCluster snapshots",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// parse vCluster config
			vConfig, err := config.ParseConfig(constants.DefaultVClusterConfigLocation, os.Getenv("VCLUSTER_NAME"), nil)
			if err != nil {
				return err
			}

			envOptions, err := snapshot.ParseOptionsFromEnv()
			if err != nil {
				return fmt.Errorf("failed to parse options from environment: %w", err)
			}

			restClient, vClusterNamespace, err := setupconfig.InitClientConfig()
			if err != nil {
				return fmt.Errorf("failed to init client config: %w", err)
			}

			kubeClient, err := kubernetes.NewForConfig(restClient)
			if err != nil {
				return fmt.Errorf("failed to create kube client: %w", err)
			}

			request, err := snapshot.CreateSnapshotRequestResources(cmd.Context(), vClusterNamespace, vConfig.Name, vConfig, envOptions, kubeClient)
			if err != nil {
				return fmt.Errorf("failed to create snapshot request resources: %w", err)
			}

			encodedBytes, err := json.Marshal(request)
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
