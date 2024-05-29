package connect

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// LoftDirectClusterEndpoint is a cluster annotation that tells the loft cli to use this endpoint instead of
	// the default loft server address to connect to this cluster.
	LoftDirectClusterEndpoint = "loft.sh/direct-cluster-endpoint"

	// LoftDirectClusterEndpointInsecure is a cluster annotation that tells the loft cli to allow untrusted certificates
	LoftDirectClusterEndpointInsecure = "loft.sh/direct-cluster-endpoint-insecure"

	// LoftDirectClusterEndpointCaData is a cluster annotation that tells the loft cli which cluster ca data to use
	LoftDirectClusterEndpointCaData = "loft.sh/direct-cluster-endpoint-ca-data"
)

// ClusterCmd holds the cmd flags
type ClusterCmd struct {
	*flags.GlobalFlags

	Print                        bool
	DisableDirectClusterEndpoint bool

	log log.Logger
}

// newClusterCmd creates a new command
func newClusterCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ClusterCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("use cluster", `
Creates a new kube context for the given cluster, if
it does not yet exist.

Example:
vcluster platform connect cluster mycluster
########################################################
	`)
	c := &cobra.Command{
		Use:   "cluster",
		Short: "Creates a kube context for the given cluster",
		Long:  description,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			if !cmd.Print {
				upgrade.PrintNewerVersionWarning()
			}

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	c.Flags().BoolVar(&cmd.Print, "print", false, "When enabled prints the context to stdout")
	return c
}

// Run executes the command
func (cmd *ClusterCmd) Run(ctx context.Context, args []string) error {
	platformClient, err := platform.NewClientFromConfig(ctx, cmd.LoadedConfig(cmd.log))
	if err != nil {
		return err
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	// determine cluster name
	clusterName := ""
	if len(args) == 0 {
		clusterName, err = platformClient.SelectCluster(ctx, cmd.log)
		if err != nil {
			return err
		}
	} else {
		clusterName = args[0]
	}

	// check if the cluster exists
	cluster, err := managementClient.Loft().ManagementV1().Clusters().Get(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsForbidden(err) {
			return fmt.Errorf("cluster '%s' does not exist, or you don't have permission to use it", clusterName)
		}

		return err
	}

	// create kube context options
	contextOptions, err := CreateClusterContextOptions(platformClient, cmd.Config, cluster, "", true)
	if err != nil {
		return err
	}

	// check if we should print or update the config
	if cmd.Print {
		err = kubeconfig.PrintKubeConfigTo(contextOptions, os.Stdout)
		if err != nil {
			return err
		}
	} else {
		// update kube config
		err = kubeconfig.UpdateKubeConfig(contextOptions)
		if err != nil {
			return err
		}

		cmd.log.Donef("Successfully updated kube context to use cluster %s", ansi.Color(clusterName, "white+b"))
	}

	return nil
}

func CreateClusterContextOptions(platformClient platform.Client, config string, cluster *managementv1.Cluster, spaceName string, setActive bool) (kubeconfig.ContextOptions, error) {
	contextOptions := kubeconfig.ContextOptions{
		Name:             kubeconfig.SpaceContextName(cluster.Name, spaceName),
		ConfigPath:       config,
		CurrentNamespace: spaceName,
		SetActive:        setActive,
	}
	contextOptions.Server = platformClient.Config().Platform.Host + "/kubernetes/cluster/" + cluster.Name
	contextOptions.InsecureSkipTLSVerify = platformClient.Config().Platform.Insecure

	data, err := retrieveCaData(cluster)
	if err != nil {
		return kubeconfig.ContextOptions{}, err
	}
	contextOptions.CaData = data
	return contextOptions, nil
}

func retrieveCaData(cluster *managementv1.Cluster) ([]byte, error) {
	if cluster == nil || cluster.Annotations == nil || cluster.Annotations[LoftDirectClusterEndpointCaData] == "" {
		return nil, nil
	}

	data, err := base64.StdEncoding.DecodeString(cluster.Annotations[LoftDirectClusterEndpointCaData])
	if err != nil {
		return nil, fmt.Errorf("error decoding cluster %s annotation: %w", LoftDirectClusterEndpointCaData, err)
	}

	return data, nil
}
