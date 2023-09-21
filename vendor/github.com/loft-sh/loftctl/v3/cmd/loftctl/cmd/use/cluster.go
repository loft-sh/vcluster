package use

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	managementv1 "github.com/loft-sh/api/v3/pkg/apis/management/v1"
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/client/helper"
	"github.com/loft-sh/loftctl/v3/pkg/kubeconfig"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/loft-sh/log"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
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

// NewClusterCmd creates a new command
func NewClusterCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ClusterCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("use cluster", `
Creates a new kube context for the given cluster, if
it does not yet exist.

Example:
loft use cluster mycluster
########################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
########################################################
################ devspace use cluster ##################
########################################################
Creates a new kube context for the given cluster, if
it does not yet exist.

Example:
devspace use cluster mycluster
########################################################
	`
	}
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
	c.Flags().BoolVar(&cmd.DisableDirectClusterEndpoint, "disable-direct-cluster-endpoint", false, "When enabled does not use an available direct cluster endpoint to connect to the cluster")
	return c
}

// Run executes the command
func (cmd *ClusterCmd) Run(ctx context.Context, args []string) error {
	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	// determine cluster name
	clusterName := ""
	if len(args) == 0 {
		clusterName, err = helper.SelectCluster(baseClient, cmd.log)
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
	contextOptions, err := CreateClusterContextOptions(baseClient, cmd.Config, cluster, "", cmd.DisableDirectClusterEndpoint, true, cmd.log)
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

func findProjectCluster(ctx context.Context, baseClient client.Client, projectName, clusterName string) (*managementv1.Cluster, error) {
	managementClient, err := baseClient.Management()
	if err != nil {
		return nil, err
	}

	projectClusters, err := managementClient.Loft().ManagementV1().Projects().ListClusters(ctx, projectName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "list project clusters")
	}

	for _, cluster := range projectClusters.Clusters {
		if cluster.Name == clusterName {
			return &cluster, nil
		}
	}

	return nil, fmt.Errorf("couldn't find cluster %s in project %s", clusterName, projectName)
}

func CreateClusterContextOptions(baseClient client.Client, config string, cluster *managementv1.Cluster, spaceName string, disableClusterGateway, setActive bool, log log.Logger) (kubeconfig.ContextOptions, error) {
	contextOptions := kubeconfig.ContextOptions{
		Name:             kubeconfig.SpaceContextName(cluster.Name, spaceName),
		ConfigPath:       config,
		CurrentNamespace: spaceName,
		SetActive:        setActive,
	}
	if !disableClusterGateway && cluster.Annotations != nil && cluster.Annotations[LoftDirectClusterEndpoint] != "" {
		contextOptions = ApplyDirectClusterEndpointOptions(contextOptions, cluster, "/kubernetes/cluster", log)
		_, err := baseClient.DirectClusterEndpointToken(true)
		if err != nil {
			return kubeconfig.ContextOptions{}, fmt.Errorf("retrieving direct cluster endpoint token: %w. Use --disable-direct-cluster-endpoint to create a context without using direct cluster endpoints", err)
		}
	} else {
		contextOptions.Server = baseClient.Config().Host + "/kubernetes/cluster/" + cluster.Name
		contextOptions.InsecureSkipTLSVerify = baseClient.Config().Insecure
	}

	data, err := retrieveCaData(cluster)
	if err != nil {
		return kubeconfig.ContextOptions{}, err
	}
	contextOptions.CaData = data
	return contextOptions, nil
}

func ApplyDirectClusterEndpointOptions(options kubeconfig.ContextOptions, cluster *managementv1.Cluster, path string, log log.Logger) kubeconfig.ContextOptions {
	server := strings.TrimSuffix(cluster.Annotations[LoftDirectClusterEndpoint], "/")
	if !strings.HasPrefix(server, "https://") {
		server = "https://" + server
	}

	log.Infof("Using direct cluster endpoint at %s", server)
	options.Server = server + path
	if cluster.Annotations[LoftDirectClusterEndpointInsecure] == "true" {
		options.InsecureSkipTLSVerify = true
	}
	options.DirectClusterEndpointEnabled = true
	return options
}

func retrieveCaData(cluster *managementv1.Cluster) ([]byte, error) {
	if cluster.Annotations == nil || cluster.Annotations[LoftDirectClusterEndpointCaData] == "" {
		return nil, nil
	}

	data, err := base64.StdEncoding.DecodeString(cluster.Annotations[LoftDirectClusterEndpointCaData])
	if err != nil {
		return nil, fmt.Errorf("error decoding cluster %s annotation: %w", LoftDirectClusterEndpointCaData, err)
	}

	return data, nil
}
