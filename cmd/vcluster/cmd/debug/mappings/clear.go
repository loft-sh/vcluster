package mappings

import (
	"context"
	"fmt"
	"os"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"github.com/loft-sh/vcluster/pkg/mappings/store"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

type ClearOptions struct {
	Config string
}

func NewClearCommand() *cobra.Command {
	options := &ClearOptions{}
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Empty the vCluster stored mappings",
		Args:  cobra.NoArgs,
		RunE: func(cobraCommand *cobra.Command, _ []string) (err error) {
			return ExecuteClear(cobraCommand.Context(), options)
		},
	}

	cmd.Flags().StringVar(&options.Config, "config", constants.DefaultVClusterConfigLocation, "The path where to find the vCluster config to load")
	return cmd
}
func ExecuteClear(ctx context.Context, options *ClearOptions) error {
	// parse vCluster config
	vConfig, err := config.ParseConfig(options.Config, os.Getenv("VCLUSTER_NAME"), nil)
	if err != nil {
		return err
	}

	// create new etcd client
	etcdClient, err := etcd.NewFromConfig(ctx, vConfig)
	if err != nil {
		return err
	}

	// create new etcd backend & list mappings
	etcdBackend := store.NewEtcdBackend(etcdClient)
	mappings, err := etcdBackend.List(ctx)
	if err != nil {
		return fmt.Errorf("list mappings: %w", err)
	}

	// print mappings
	for _, mapping := range mappings {
		klog.FromContext(ctx).Info("Delete mapping", "mapping", mapping.String())
		err = etcdBackend.Delete(ctx, mapping)
		if err != nil {
			return fmt.Errorf("delete mapping %s: %w", mapping.String(), err)
		}
	}

	return nil
}
