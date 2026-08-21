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

	Kind string
}

func NewListCommand() *cobra.Command {
	options := &ListOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Dump the vCluster stored mappings",
		Args:  cobra.NoArgs,
		RunE: func(cobraCommand *cobra.Command, _ []string) (err error) {
			return ExecuteList(cobraCommand.Context(), options)
		},
	}

	cmd.Flags().StringVar(&options.Config, "config", constants.DefaultVClusterConfigLocation, "The path where to find the vCluster config to load")
	cmd.Flags().StringVar(&options.Kind, "kind", "", "The kind of objects to list")
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

	// filter if kind is specified
	if options.Kind != "" {
		newMappings := make([]*store.Mapping, 0, len(mappings))
		for _, mapping := range mappings {
			if mapping.Kind != options.Kind {
				continue
			}

			newMappings = append(newMappings, mapping)
		}
		mappings = newMappings
	}

	// print mappings
	raw, err := json.MarshalIndent(mappings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal mappings: %w", err)
	}

	fmt.Println(string(raw))
	return nil
}
