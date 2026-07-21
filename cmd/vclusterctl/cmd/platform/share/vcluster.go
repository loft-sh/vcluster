package share

import (
	"context"
	"fmt"

	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/projectutil"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VClusterCmd holds the login cmd flags
type VClusterCmd struct {
	*flags.GlobalFlags

	Project     string
	Cluster     string
	ClusterRole string
	User        string
	Team        string

	Log log.Logger
}

// NewVClustersCmd creates a new command
func NewVClusterCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &VClusterCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "vcluster" + util.VClusterNameOnlyUseLine,
		Short: "Shares a virtual cluster with another vCluster platform user or team",
		Long: `##########################################################################
#################### vcluster platform share vcluster ####################
##########################################################################
Shares a virtual cluster with another vCluster platform user or team

Example:
vcluster platform share vcluster myvcluster
vcluster platform share vcluster myvcluster --project myproject
vcluster platform share vcluster myvcluster --project myproject --user admin
##########################################################################
	`,
		Args: util.VClusterNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Cluster, "cluster", "", "The cluster to use")
	cobraCmd.Flags().StringVar(&cmd.Project, "project", "", "The project to use")
	cobraCmd.Flags().StringVar(&cmd.ClusterRole, "cluster-role", "loft-cluster-space-admin", "The cluster role which is assigned to the user or team for that space")
	cobraCmd.Flags().StringVar(&cmd.User, "user", "", "The user to share the space with. The user needs to have access to the cluster")
	cobraCmd.Flags().StringVar(&cmd.Team, "team", "", "The team to share the space with. The team needs to have access to the cluster")

	return cobraCmd
}

// TODO(johannesfrey): Move flags and functionality to pkg/cli
func (cmd *VClusterCmd) Run(ctx context.Context, args []string) error {
	vClusterName := args[0]

	platformClient, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.Log))
	if err != nil {
		return err
	}

	// determine project & cluster name
	cmd.Cluster, cmd.Project, err = platform.SelectProjectOrCluster(ctx, platformClient, cmd.Cluster, cmd.Project, false, cmd.Log)
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

	accessRule := storagev1.InstanceAccessRule{
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
		cmd.Log.Donef("Successfully granted user %s access to vcluster %s", ansi.Color(cmd.User, "white+b"), ansi.Color(vClusterName, "white+b"))
		cmd.Log.Infof("The user can access the space now via: %s", ansi.Color(fmt.Sprintf(product.Replace("vcluster connect %s --project %s"), vClusterName, cmd.Project), "white+b"))
	} else {
		cmd.Log.Donef("Successfully granted team %s access to vcluster %s", ansi.Color(cmd.Team, "white+b"), ansi.Color(vClusterName, "white+b"))
		cmd.Log.Infof("The team can access the space now via: %s", ansi.Color(fmt.Sprintf(product.Replace("vcluster connect %s --project %s"), vClusterName, cmd.Project), "white+b"))
	}

	return nil
}
