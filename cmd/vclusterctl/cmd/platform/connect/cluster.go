package connect

import (
	"context"
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

// ClusterCmd holds the cmd flags
type ClusterCmd struct {
	log log.Logger
	*flags.GlobalFlags

	Print                        bool
	DisableDirectClusterEndpoint bool
}

// newClusterCmd creates a new command
func newClusterCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ClusterCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("connect cluster", `
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
	cfg := cmd.LoadedConfig(cmd.log)
	platformClient, err := platform.InitClientFromConfig(ctx, cfg)
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
		clusterName, err = platform.SelectCluster(ctx, platformClient, cmd.log)
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
		err = kubeconfig.UpdateKubeConfig(contextOptions, cfg)
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

	data, err := platform.RetrieveCaData(cluster)
	if err != nil {
		return kubeconfig.ContextOptions{}, err
	}
	contextOptions.CaData = data
	return contextOptions, nil
}
