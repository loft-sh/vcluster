package etcd

import (
	"context"
	"fmt"
	"os"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"github.com/spf13/cobra"
)

type KeysOptions struct {
	Config string

	Prefix string
}

func NewKeysCommand() *cobra.Command {
	options := &KeysOptions{}
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Dump the vCluster etcd stored keys",
		Args:  cobra.NoArgs,
		RunE: func(cobraCommand *cobra.Command, _ []string) (err error) {
			return ExecuteKeys(cobraCommand.Context(), options)
		},
	}

	cmd.Flags().StringVar(&options.Config, "config", constants.DefaultVClusterConfigLocation, "The path where to find the vCluster config to load")
	cmd.Flags().StringVar(&options.Prefix, "prefix", "/", "The prefix to use for listing the keys")
	return cmd
}

func ExecuteKeys(ctx context.Context, options *KeysOptions) error {
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
	keyValues, err := etcdClient.List(ctx, options.Prefix)
	if err != nil {
		return err
	}

	// print mappings
	for _, keyValue := range keyValues {
		fmt.Println(string(keyValue.Key))
	}

	return nil
}
