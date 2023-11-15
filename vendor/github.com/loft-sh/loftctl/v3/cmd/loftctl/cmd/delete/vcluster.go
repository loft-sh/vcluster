package delete

import (
	"context"
	"time"

	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/pkg/client/naming"
	"github.com/loft-sh/loftctl/v3/pkg/constants"
	"github.com/loft-sh/loftctl/v3/pkg/kube"
	"github.com/loft-sh/loftctl/v3/pkg/util"

	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/client/helper"
	pdefaults "github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/loft-sh/loftctl/v3/pkg/kubeconfig"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/loft-sh/log"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VirtualClusterCmd holds the cmd flags
type VirtualClusterCmd struct {
	*flags.GlobalFlags

	Space         string
	Cluster       string
	Project       string
	DeleteContext bool
	DeleteSpace   bool
	Wait          bool

	Log log.Logger
}

// NewVirtualClusterCmd creates a new command
func NewVirtualClusterCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &VirtualClusterCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("delete virtualcluster", `
Deletes a virtual cluster from a cluster

Example:
loft delete vcluster myvirtualcluster
loft delete vcluster myvirtualcluster --project myproject
########################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
#######################################################
########## devspace delete virtualcluster #############
#######################################################
Deletes a virtual cluster from a cluster

Example:
devspace delete vcluster myvirtualcluster
devspace delete vcluster myvirtualcluster --project myproject
#######################################################
	`
	}
	c := &cobra.Command{
		Use:   "vcluster" + util.VClusterNameOnlyUseLine,
		Short: "Deletes a virtual cluster from a cluster",
		Long:  description,
		Args:  util.VClusterNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVar(&cmd.Space, "space", "", "The space to use")
	c.Flags().StringVar(&cmd.Cluster, "cluster", "", "The cluster to use")
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project to use")
	c.Flags().BoolVar(&cmd.DeleteContext, "delete-context", true, "If the corresponding kube context should be deleted if there is any")
	c.Flags().BoolVar(&cmd.DeleteSpace, "delete-space", false, "Should the corresponding space be deleted")
	c.Flags().BoolVar(&cmd.Wait, "wait", false, "Termination of this command waits for space to be deleted. Without the flag delete-space, this flag has no effect.")
	return c
}

// Run executes the command
func (cmd *VirtualClusterCmd) Run(ctx context.Context, args []string) error {
	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	virtualClusterName := ""
	if len(args) > 0 {
		virtualClusterName = args[0]
	}

	cmd.Cluster, cmd.Project, cmd.Space, virtualClusterName, err = helper.SelectVirtualClusterInstanceOrVirtualCluster(baseClient, virtualClusterName, cmd.Space, cmd.Project, cmd.Cluster, cmd.Log)
	if err != nil {
		return err
	}

	if cmd.Project == "" {
		return cmd.legacyDeleteVirtualCluster(ctx, baseClient, virtualClusterName)
	}

	return cmd.deleteVirtualCluster(ctx, baseClient, virtualClusterName)
}

func (cmd *VirtualClusterCmd) deleteVirtualCluster(ctx context.Context, baseClient client.Client, virtualClusterName string) error {
	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	err = managementClient.Loft().ManagementV1().VirtualClusterInstances(naming.ProjectNamespace(cmd.Project)).Delete(ctx, virtualClusterName, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "delete virtual cluster")
	}

	cmd.Log.Donef("Successfully deleted virtual cluster %s in project %s", ansi.Color(virtualClusterName, "white+b"), ansi.Color(cmd.Project, "white+b"))

	// update kube config
	if cmd.DeleteContext {
		err = kubeconfig.DeleteContext(kubeconfig.VirtualClusterInstanceContextName(cmd.Project, virtualClusterName))
		if err != nil {
			return err
		}

		cmd.Log.Donef("Successfully deleted kube context for virtual cluster %s", ansi.Color(virtualClusterName, "white+b"))
	}

	// wait until deleted
	if cmd.Wait {
		cmd.Log.Info("Waiting for virtual cluster to be deleted...")
		for isVirtualClusterInstanceStillThere(ctx, managementClient, naming.ProjectNamespace(cmd.Project), virtualClusterName) {
			time.Sleep(time.Second)
		}
		cmd.Log.Done("Virtual Cluster is deleted")
	}

	return nil
}

func isVirtualClusterInstanceStillThere(ctx context.Context, managementClient kube.Interface, namespace, name string) bool {
	_, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(namespace).Get(ctx, name, metav1.GetOptions{})
	return err == nil
}

func (cmd *VirtualClusterCmd) legacyDeleteVirtualCluster(ctx context.Context, baseClient client.Client, virtualClusterName string) error {
	clusterClient, err := baseClient.Cluster(cmd.Cluster)
	if err != nil {
		return err
	}

	gracePeriod := int64(0)
	err = clusterClient.Agent().StorageV1().VirtualClusters(cmd.Space).Delete(ctx, virtualClusterName, metav1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
	if err != nil {
		return errors.Wrap(err, "delete virtual cluster")
	}

	cmd.Log.Donef("Successfully deleted virtual cluster %s in space %s in cluster %s", ansi.Color(virtualClusterName, "white+b"), ansi.Color(cmd.Space, "white+b"), ansi.Color(cmd.Cluster, "white+b"))

	// update kube config
	if cmd.DeleteContext {
		err = kubeconfig.DeleteContext(kubeconfig.VirtualClusterContextName(cmd.Cluster, cmd.Space, virtualClusterName))
		if err != nil {
			return err
		}

		cmd.Log.Donef("Successfully deleted kube context for virtual cluster %s", ansi.Color(virtualClusterName, "white+b"))
	}

	// check if we should delete space
	spaceObject, err := clusterClient.Agent().ClusterV1().Spaces().Get(ctx, cmd.Space, metav1.GetOptions{})
	if err == nil && spaceObject.Annotations != nil && spaceObject.Annotations[constants.VClusterSpace] == "true" {
		cmd.DeleteSpace = true
	}

	// delete space
	if cmd.DeleteSpace {
		err = clusterClient.Agent().ClusterV1().Spaces().Delete(ctx, cmd.Space, metav1.DeleteOptions{})
		if err != nil {
			return err
		}

		// wait for termination
		if cmd.Wait {
			cmd.Log.Info("Waiting for space to be deleted...")
			for isSpaceStillThere(ctx, clusterClient, cmd.Space) {
				time.Sleep(time.Second)
			}
		}

		cmd.Log.Donef("Successfully deleted space %s", cmd.Space)
	}

	return nil
}
