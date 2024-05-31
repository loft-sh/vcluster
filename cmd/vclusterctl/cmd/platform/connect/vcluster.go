package connect

import (
	"context"
	"fmt"

	agentstoragev1 "github.com/loft-sh/agentapi/v4/pkg/apis/loft/storage/v1"
	"github.com/loft-sh/api/v4/pkg/product"
	loftctlUtil "github.com/loft-sh/loftctl/v4/pkg/util"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/projectutil"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VClusterCmd holds the cmd flags
type VClusterCmd struct {
	*flags.GlobalFlags

	Cluster     string
	Project     string
	ClusterRole string
	User        string
	Team        string

	log log.Logger
}

// newVClusterCmd creates a new command
func newVClusterCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &VClusterCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("platform connect vcluster", `
Creates a kube context for the given virtual cluster

Example:
vcluster platform connect vcluster myvcluster
########################################################
	`)
	c := &cobra.Command{
		Use:   "vcluster" + loftctlUtil.VClusterNameOnlyUseLine,
		Short: "Creates a kube context for the given virtual cluster",
		Long:  description,
		Args:  loftctlUtil.VClusterNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	c.Flags().StringVar(&cmd.Cluster, "cluster", "", "[PLATFORM] The cluster to use")
	c.Flags().StringVar(&cmd.Project, "project", "", "[PLATFORM] The project to use")
	c.Flags().StringVar(&cmd.ClusterRole, "cluster-role", "loft-cluster-space-admin", "[PLATFORM] The cluster role which is assigned to the user or team for that space")
	c.Flags().StringVar(&cmd.User, "user", "", "[PLATFORM] The user to share the space with. The user needs to have access to the cluster")
	c.Flags().StringVar(&cmd.Team, "team", "", "[PLATFORM] The team to share the space with. The team needs to have access to the cluster")
	return c
}

// Run executes the command
// TODO(johannesfrey): Move flags and functionality to pkg/cli
func (cmd *VClusterCmd) Run(ctx context.Context, args []string) error {
	vClusterName := args[0]

	platformClient, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.log))
	if err != nil {
		return err
	}

	if err := platform.VerifyVersion(platformClient); err != nil {
		return err
	}

	// determine project & cluster name
	cmd.Cluster, cmd.Project, err = platform.SelectProjectOrCluster(ctx, platformClient, cmd.Cluster, cmd.Project, false, cmd.log)
	if err != nil {
		return err
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	virtualClusterInstance, err := managementClient.Loft().ManagementV1().VirtualClusterInstances(projectutil.ProjectNamespace(cmd.Project)).Get(ctx, vClusterName, metav1.GetOptions{})
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
	_, err = managementClient.Loft().ManagementV1().VirtualClusterInstances(projectutil.ProjectNamespace(cmd.Project)).Update(ctx, virtualClusterInstance, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	if cmd.User != "" {
		cmd.log.Donef("Successfully granted user %s access to vcluster %s", ansi.Color(cmd.User, "white+b"), ansi.Color(vClusterName, "white+b"))
		cmd.log.Infof("The user can access the space now via: %s", ansi.Color(fmt.Sprintf(product.Replace("loft use vcluster %s --project %s"), vClusterName, cmd.Project), "white+b"))
	} else {
		cmd.log.Donef("Successfully granted team %s access to vcluster %s", ansi.Color(cmd.Team, "white+b"), ansi.Color(vClusterName, "white+b"))
		cmd.log.Infof("The team can access the space now via: %s", ansi.Color(fmt.Sprintf(product.Replace("loft use vcluster %s --project %s"), vClusterName, cmd.Project), "white+b"))
	}

	return nil
}
