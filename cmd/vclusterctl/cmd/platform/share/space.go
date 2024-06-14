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
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/loft-sh/vcluster/pkg/projectutil"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NamespaceCmd holds the cmd flags
type NamespaceCmd struct {
	*flags.GlobalFlags

	Project     string
	Cluster     string
	ClusterRole string
	User        string
	Team        string

	Log log.Logger
}

// NewNamespaceCmd creates a new command
func NewNamespaceCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &NamespaceCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("share namespace", `
Shares a vCluster platform namespace with another platform user or team. The user
or team need to have access to the cluster.
Example:
vcluster platform share namespace myspace
vcluster platform share namespace myspace --project myproject
vcluster platform share namespace myspace --project myproject --user admin
########################################################
	`)
	c := &cobra.Command{
		Use:   "namespace" + util.NamespaceNameOnlyUseLine,
		Short: product.Replace("Shares a vCluster platform namespace with another platform user or team"),
		Long:  description,
		Args:  util.NamespaceNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVar(&cmd.Cluster, "cluster", "", "The cluster to use")
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project to use")
	c.Flags().StringVar(&cmd.ClusterRole, "cluster-role", "loft-cluster-space-admin", "The cluster role which is assigned to the user or team for that namespace")
	c.Flags().StringVar(&cmd.User, "user", "", "The user to share the namespace with. The user needs to have access to the cluster")
	c.Flags().StringVar(&cmd.Team, "team", "", "The team to share the namespace with. The team needs to have access to the cluster")
	return c
}

// Run executes the command
func (cmd *NamespaceCmd) Run(ctx context.Context, args []string) error {
	platformClient, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.Log))
	if err != nil {
		return err
	}

	spaceName := ""
	if len(args) > 0 {
		spaceName = args[0]
	}

	cmd.Cluster, cmd.Project, spaceName, err = platform.SelectSpaceInstance(ctx, platformClient, spaceName, cmd.Project, cmd.Log)
	if err != nil {
		return err
	}

	return cmd.shareSpace(ctx, platformClient, spaceName)
}

func (cmd *NamespaceCmd) shareSpace(ctx context.Context, platformClient platform.Client, spaceName string) error {
	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	spaceInstance, err := managementClient.Loft().ManagementV1().SpaceInstances(projectutil.ProjectNamespace(cmd.Project)).Get(ctx, spaceName, metav1.GetOptions{})
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
	spaceInstance.Spec.ExtraAccessRules = append(spaceInstance.Spec.ExtraAccessRules, accessRule)
	if spaceInstance.Spec.TemplateRef != nil {
		spaceInstance.Spec.TemplateRef.SyncOnce = true
	}
	_, err = managementClient.Loft().ManagementV1().SpaceInstances(projectutil.ProjectNamespace(cmd.Project)).Update(ctx, spaceInstance, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	if cmd.User != "" {
		cmd.Log.Donef("Successfully granted user %s access to namespace %s", ansi.Color(cmd.User, "white+b"), ansi.Color(spaceName, "white+b"))
		cmd.Log.Infof("The user can access the namespace now via: %s", ansi.Color(fmt.Sprintf(product.Replace("vcluster platform connect namespace %s --project %s"), spaceName, cmd.Project), "white+b"))
	} else {
		cmd.Log.Donef("Successfully granted team %s access to namespace %s", ansi.Color(cmd.Team, "white+b"), ansi.Color(spaceName, "white+b"))
		cmd.Log.Infof("The team can access the namespace now via: %s", ansi.Color(fmt.Sprintf(product.Replace("vcluster platform connect namespace %s --project %s"), spaceName, cmd.Project), "white+b"))
	}

	return nil
}
