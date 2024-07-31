package mappings

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/etcd"
	"github.com/loft-sh/vcluster/pkg/mappings/store"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
)

type AddOptions struct {
	Config string

	APIVersion string
	Kind       string

	Host    string
	Virtual string
}

func NewAddCommand() *cobra.Command {
	options := &AddOptions{}
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Adds a custom mapping to the vCluster stored mappings",
		RunE: func(cobraCommand *cobra.Command, _ []string) (err error) {
			return ExecuteSave(cobraCommand.Context(), options)
		},
	}

	cmd.Flags().StringVar(&options.Config, "config", constants.DefaultVClusterConfigLocation, "The path where to find the vCluster config to load")
	cmd.Flags().StringVar(&options.Kind, "kind", "", "The Kind of the object")
	cmd.Flags().StringVar(&options.APIVersion, "api-version", "", "The APIVersion of the object")
	cmd.Flags().StringVar(&options.Host, "host", "", "The host object in the form of namespace/name")
	cmd.Flags().StringVar(&options.Virtual, "virtual", "", "The virtual object in the form of namespace/name")

	return cmd
}
func ExecuteSave(ctx context.Context, options *AddOptions) error {
	nameMapping, etcdBackend, err := parseMappingAndClient(ctx, options.Config, options.Kind, options.APIVersion, options.Virtual, options.Host)
	if err != nil {
		return err
	}

	err = etcdBackend.Save(ctx, &store.Mapping{
		NameMapping: nameMapping,
	})
	if err != nil {
		return fmt.Errorf("error saving %s: %w", nameMapping.String(), err)
	}

	klog.FromContext(ctx).Info("Successfully added name mapping to store", "mapping", nameMapping.String())
	return nil
}

func parseMappingAndClient(ctx context.Context, configPath, kind, apiVersion, virtual, host string) (synccontext.NameMapping, store.Backend, error) {
	if kind == "" || apiVersion == "" || virtual == "" || host == "" {
		return synccontext.NameMapping{}, nil, fmt.Errorf("make sure to specify --kind, --api-version, --host and --virtual")
	}

	// parse group version
	groupVersion, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return synccontext.NameMapping{}, nil, fmt.Errorf("parse group version: %w", err)
	}

	// parse host
	hostName := types.NamespacedName{Name: host}
	if strings.Contains(host, "/") {
		namespaceName := strings.SplitN(host, "/", 2)
		hostName.Namespace = namespaceName[0]
		hostName.Name = namespaceName[1]
	}

	// parse virtual
	virtualName := types.NamespacedName{Name: virtual}
	if strings.Contains(virtual, "/") {
		namespaceName := strings.SplitN(virtual, "/", 2)
		virtualName.Namespace = namespaceName[0]
		virtualName.Name = namespaceName[1]
	}

	// build name mapping
	nameMapping := synccontext.NameMapping{
		GroupVersionKind: schema.GroupVersionKind{
			Group:   groupVersion.Group,
			Version: groupVersion.Version,
			Kind:    kind,
		},
		VirtualName: virtualName,
		HostName:    hostName,
	}

	// parse vCluster config
	vConfig, err := config.ParseConfig(configPath, os.Getenv("VCLUSTER_NAME"), nil)
	if err != nil {
		return synccontext.NameMapping{}, nil, err
	}

	// create new etcd client
	etcdClient, err := etcd.NewFromConfig(ctx, vConfig)
	if err != nil {
		return synccontext.NameMapping{}, nil, err
	}

	// create new etcd backend & list mappings
	etcdBackend := store.NewEtcdBackend(etcdClient)
	return nameMapping, etcdBackend, nil
}
