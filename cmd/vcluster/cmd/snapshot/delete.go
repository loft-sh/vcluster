package snapshot

import (
	"fmt"
	"os"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	setupconfig "github.com/loft-sh/vcluster/pkg/setup/config"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

func NewDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "delete vCluster snapshot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
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

			err = snapshot.DeleteSnapshotRequestResources(cmd.Context(), vClusterNamespace, vConfig.Name, vConfig, envOptions, kubeClient)
			if err != nil {
				return fmt.Errorf("failed to delete snapshot request resources: %w", err)
			}

			return nil
		},
	}

	return cmd
}
