package snapshot

import (
	"fmt"
	"os"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get vCluster snapshot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			vClusterName := os.Getenv("VCLUSTER_NAME")
			snapshotOpts, err := snapshot.ParseOptionsFromEnv()
			if err != nil {
				return fmt.Errorf("failed to parse options from environment: %w", err)
			}
			kubeClientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
			restConfig, err := kubeClientConfig.ClientConfig()
			if err != nil {
				return fmt.Errorf("failed to get kube client config: %w", err)
			}
			kubeClient, err := kubernetes.NewForConfig(restConfig)
			if err != nil {
				return fmt.Errorf("failed to create kube client: %w", err)
			}
			logger := log.GetInstance()

			err = snapshot.GetSnapshots(ctx, vClusterName, snapshotOpts, kubeClient, logger)
			if err != nil {
				return fmt.Errorf("failed to list snapshots: %w", err)
			}
			return nil
		},
	}

	return cmd
}
