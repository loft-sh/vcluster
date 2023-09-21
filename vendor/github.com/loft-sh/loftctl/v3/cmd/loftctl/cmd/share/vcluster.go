package share

import (
	"context"
	"fmt"

	agentstoragev1 "github.com/loft-sh/agentapi/v3/pkg/apis/loft/storage/v1"
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/client/helper"
	"github.com/loft-sh/loftctl/v3/pkg/client/naming"
	pdefaults "github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/loft-sh/loftctl/v3/pkg/util"
	"github.com/loft-sh/log"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VClusterCmd holds the cmd flags
type VClusterCmd struct {
	*flags.GlobalFlags

	Project     string
	Cluster     string
	Space       string
	ClusterRole string
	User        string
	Team        string

	Log log.Logger
}

// NewVClusterCmd creates a new command
func NewVClusterCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &VClusterCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("share vcluster", `
Shares a vcluster with another loft user or team. The
user or team need to have access to the cluster.

Example:
loft share vcluster myvcluster
loft share vcluster myvcluster --cluster mycluster
loft share vcluster myvcluster --cluster mycluster --user admin
########################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
########################################################
############### devspace share vcluster ################
########################################################
Shares a vcluster with another loft user or team. The
user or team need to have access to the cluster.

Example:
devspace share vcluster myvcluster
devspace share vcluster myvcluster --project myproject
devspace share vcluster myvcluster --project myproject --user admin
########################################################
	`
	}
	c := &cobra.Command{
		Use:   "vcluster" + util.VClusterNameOnlyUseLine,
		Short: product.Replace("Shares a vcluster with another loft user or team"),
		Long:  description,
		Args:  util.VClusterNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd, args)
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVar(&cmd.Cluster, "cluster", "", "The cluster to use")
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project to use")
	c.Flags().StringVar(&cmd.Space, "space", "", "The space to use")
	c.Flags().StringVar(&cmd.ClusterRole, "cluster-role", "loft-cluster-space-admin", "The cluster role which is assigned to the user or team for that space")
	c.Flags().StringVar(&cmd.User, "user", "", "The user to share the space with. The user needs to have access to the cluster")
	c.Flags().StringVar(&cmd.Team, "team", "", "The team to share the space with. The team needs to have access to the cluster")
	return c
}

// Run executes the command
func (cmd *VClusterCmd) Run(cobraCmd *cobra.Command, args []string) error {
	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	vClusterName := ""
	if len(args) > 0 {
		vClusterName = args[0]
	}

	cmd.Cluster, cmd.Project, cmd.Space, vClusterName, err = helper.SelectVirtualClusterInstanceOrVirtualCluster(baseClient, vClusterName, cmd.Space, cmd.Project, cmd.Cluster, cmd.Log)
	if err != nil {
		return err
	}

	ctx := cobraCmd.Context()

	if cmd.Project == "" {
		return cmd.legacyShareVCluster(ctx, baseClient, vClusterName)
	}

	return cmd.shareVCluster(ctx, baseClient, vClusterName)
}

func (cmd *VClusterCmd) shareVCluster(ctx context.Context, baseClient client.Client, vClusterName string) error {
	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	virtualClusterInstance, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(naming.ProjectNamespace(cmd.Project)).Get(ctx, vClusterName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	accessRule := agentstoragev1.InstanceAccessRule{
		ClusterRole: cmd.ClusterRole,
	}
	if cmd.User != "" {
		accessRule.Users = append(accessRule.Users, cmd.User)
	}
	if cmd.Team != "" {
		accessRule.Teams = append(accessRule.Teams, cmd.Team)
	}
	virtualClusterInstance.Spec.ExtraAccessRules = append(virtualClusterInstance.Spec.ExtraAccessRules, accessRule)
	if virtualClusterInstance.Spec.TemplateRef != nil {
		virtualClusterInstance.Spec.TemplateRef.SyncOnce = true
	}
	_, err = managementClient.Loft().ManagementV1().VirtualClusterInstances(naming.ProjectNamespace(cmd.Project)).Update(ctx, virtualClusterInstance, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	if cmd.User != "" {
		cmd.Log.Donef("Successfully granted user %s access to vcluster %s", ansi.Color(cmd.User, "white+b"), ansi.Color(vClusterName, "white+b"))
		cmd.Log.Infof("The user can access the space now via: %s", ansi.Color(fmt.Sprintf(product.Replace("loft use vcluster %s --project %s"), vClusterName, cmd.Project), "white+b"))
	} else {
		cmd.Log.Donef("Successfully granted team %s access to vcluster %s", ansi.Color(cmd.Team, "white+b"), ansi.Color(vClusterName, "white+b"))
		cmd.Log.Infof("The team can access the space now via: %s", ansi.Color(fmt.Sprintf(product.Replace("loft use vcluster %s --project %s"), vClusterName, cmd.Project), "white+b"))
	}

	return nil
}

func (cmd *VClusterCmd) legacyShareVCluster(ctx context.Context, baseClient client.Client, vClusterName string) error {
	userOrTeam, err := createRoleBinding(ctx, baseClient, cmd.Cluster, cmd.Space, cmd.User, cmd.Team, cmd.ClusterRole, cmd.Log)
	if err != nil {
		return err
	}

	if !userOrTeam.Team {
		cmd.Log.Donef("Successfully granted user %s access to vcluster %s", ansi.Color(userOrTeam.ClusterMember.Info.Name, "white+b"), ansi.Color(vClusterName, "white+b"))
		cmd.Log.Infof("The user can access the vcluster now via: %s", ansi.Color(fmt.Sprintf(product.Replace("loft use vcluster %s --space %s --cluster %s"), vClusterName, cmd.Space, cmd.Cluster), "white+b"))
	} else {
		cmd.Log.Donef("Successfully granted team %s access to vcluster %s", ansi.Color(userOrTeam.ClusterMember.Info.Name, "white+b"), ansi.Color(vClusterName, "white+b"))
		cmd.Log.Infof("The team can access the vcluster now via: %s", ansi.Color(fmt.Sprintf(product.Replace("loft use vcluster %s --space %s --cluster %s"), vClusterName, cmd.Space, cmd.Cluster), "white+b"))
	}

	return nil
}
