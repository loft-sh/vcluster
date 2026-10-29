package mappings

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/mappings/store"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

type DeleteOptions struct {
	Config string

	APIVersion string
	Kind       string

	Host    string
	Virtual string
}

func NewDeleteCommand() *cobra.Command {
	options := &DeleteOptions{}
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes a custom mapping to the vCluster stored mappings",
		RunE: func(cobraCommand *cobra.Command, _ []string) (err error) {
			return ExecuteDelete(cobraCommand.Context(), options)
		},
	}

	cmd.Flags().StringVar(&options.Config, "config", constants.DefaultVClusterConfigLocation, "The path where to find the vCluster config to load")
	cmd.Flags().StringVar(&options.Kind, "kind", "", "The Kind of the object")
	cmd.Flags().StringVar(&options.APIVersion, "api-version", "", "The APIVersion of the object")
	cmd.Flags().StringVar(&options.Host, "host", "", "The host object in the form of namespace/name")
	cmd.Flags().StringVar(&options.Virtual, "virtual", "", "The virtual object in the form of namespace/name")

	return cmd
}
func ExecuteDelete(ctx context.Context, options *DeleteOptions) error {
	nameMapping, etcdBackend, err := parseMappingAndClient(ctx, options.Config, options.Kind, options.APIVersion, options.Virtual, options.Host)
	if err != nil {
		return err
	}

	err = etcdBackend.Delete(ctx, &store.Mapping{
		NameMapping: nameMapping,
	})
	if err != nil {
		return fmt.Errorf("error saving %s: %w", nameMapping.String(), err)
	}

	klog.FromContext(ctx).Info("Successfully deleted name mapping from store", "mapping", nameMapping.String())
	return nil
}
