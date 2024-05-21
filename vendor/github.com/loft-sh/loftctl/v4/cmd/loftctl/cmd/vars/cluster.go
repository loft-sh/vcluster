package vars

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/loftctl/v4/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v4/pkg/client"
	"github.com/loft-sh/loftctl/v4/pkg/config"
	"github.com/loft-sh/loftctl/v4/pkg/projectutil"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

var (
	ErrNotLoftContext = errors.New("current context is not a loft context, but predefined var LOFT_CLUSTER is used")
)

type clusterCmd struct {
	*flags.GlobalFlags
}

func newClusterCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &clusterCmd{
		GlobalFlags: globalFlags,
	}

	return &cobra.Command{
		Use:   "cluster",
		Short: "Prints the current cluster",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}
}

// Run executes the command logic
func (c *clusterCmd) Run(ctx context.Context, _ []string) error {
	kubeConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).RawConfig()
	if err != nil {
		return err
	}

	kubeContext := os.Getenv("DEVSPACE_PLUGIN_KUBE_CONTEXT_FLAG")
	if kubeContext == "" {
		kubeContext = kubeConfig.CurrentContext
	}

	cluster, ok := kubeConfig.Clusters[kubeContext]
	if !ok {
		return ErrNotLoftContext
	}

	baseClient, err := client.NewClientFromPath(c.Config)
	if err != nil {
		return err
	}
	self, err := baseClient.GetSelf(ctx)
	if err != nil {
		return fmt.Errorf("failed to get self: %w", err)
	}
	projectutil.SetProjectNamespacePrefix(self.Status.ProjectNamespacePrefix)

	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}
	isProject, projectName := isProjectContext(cluster)
	if isProject {
		if isSpace, spaceName := isSpaceContext(cluster); isSpace {
			var spaceInstance *managementv1.SpaceInstance
			err := wait.PollUntilContextTimeout(ctx, time.Second, config.Timeout(), true, func(ctx context.Context) (bool, error) {
				var err error

				spaceInstance, err = managementClient.Loft().ManagementV1().SpaceInstances(projectutil.ProjectNamespace(projectName)).Get(ctx, spaceName, metav1.GetOptions{})
				if err != nil {
					return false, err
				}

				// Wait for space instance to be scheduled
				if spaceInstance.Spec.ClusterRef.Cluster == "" {
					return false, nil
				}

				return true, nil
			})
			if err != nil {
				return err
			}

			_, err = os.Stdout.Write([]byte(spaceInstance.Spec.ClusterRef.Cluster))
			return err
		}

		if isVirtualCluster, virtualClusterName := isVirtualClusterContext(cluster); isVirtualCluster {
			var virtualClusterInstance *managementv1.VirtualClusterInstance
			err := wait.PollUntilContextTimeout(ctx, time.Second, config.Timeout(), true, func(ctx context.Context) (bool, error) {
				var err error

				virtualClusterInstance, err = managementClient.Loft().ManagementV1().VirtualClusterInstances(projectutil.ProjectNamespace(projectName)).Get(ctx, virtualClusterName, metav1.GetOptions{})
				if err != nil {
					return false, err
				}

				// Wait for space instance to be scheduled
				if virtualClusterInstance.Spec.ClusterRef.Cluster == "" {
					return false, nil
				}

				return true, nil
			})
			if err != nil {
				return err
			}

			_, err = os.Stdout.Write([]byte(virtualClusterInstance.Spec.ClusterRef.Cluster))
			return err
		}

		return ErrNotLoftContext
	}

	server := strings.TrimSuffix(cluster.Server, "/")
	splitted := strings.Split(server, "/")
	if len(splitted) < 3 {
		return ErrNotLoftContext
	} else if splitted[len(splitted)-2] != "cluster" || splitted[len(splitted)-3] != "kubernetes" {
		return ErrNotLoftContext
	}

	_, err = os.Stdout.Write([]byte(splitted[len(splitted)-1]))
	return err
}

func isProjectContext(cluster *api.Cluster) (bool, string) {
	server := strings.TrimSuffix(cluster.Server, "/")
	splitted := strings.Split(server, "/")

	if len(splitted) < 8 {
		return false, ""
	}

	if splitted[4] == "project" {
		return true, splitted[5]
	}

	return false, ""
}

func isSpaceContext(cluster *api.Cluster) (bool, string) {
	server := strings.TrimSuffix(cluster.Server, "/")
	splitted := strings.Split(server, "/")

	if len(splitted) < 8 {
		return false, ""
	}

	if splitted[6] == "space" {
		return true, splitted[7]
	}

	return false, ""
}

func isVirtualClusterContext(cluster *api.Cluster) (bool, string) {
	server := strings.TrimSuffix(cluster.Server, "/")
	splitted := strings.Split(server, "/")

	if len(splitted) < 8 {
		return false, ""
	}

	if splitted[6] == "virtualcluster" {
		return true, splitted[7]
	}

	return false, ""
}
