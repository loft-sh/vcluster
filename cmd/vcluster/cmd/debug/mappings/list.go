package mappings

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"github.com/loft-sh/vcluster/pkg/mappings/store"
	"github.com/spf13/cobra"
)

type ListOptions struct {
	Config string
}

func NewListCommand() *cobra.Command {
	options := &ListOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Dump the vCluster stored mappings",
		Args:  cobra.NoArgs,
		RunE: func(cobraCommand *cobra.Command, args []string) (err error) {
			return ExecuteList(cobraCommand.Context(), options)
		},
	}

	cmd.Flags().StringVar(&options.Config, "config", constants.DefaultVClusterConfigLocation, "The path where to find the vCluster config to load")
	return cmd
}
func ExecuteList(ctx context.Context, options *ListOptions) error {
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
	mappings, err := store.NewEtcdBackend(etcdClient).List(ctx)
	if err != nil {
		return fmt.Errorf("list mappings: %w", err)
	}

	// print mappings
	raw, err := json.MarshalIndent(mappings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal mappings: %w", err)
	}

	fmt.Println(string(raw))
	return nil
}
