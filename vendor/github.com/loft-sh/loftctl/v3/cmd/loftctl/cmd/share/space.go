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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SpaceCmd holds the cmd flags
type SpaceCmd struct {
	*flags.GlobalFlags

	Project     string
	Cluster     string
	ClusterRole string
	User        string
	Team        string

	Log log.Logger
}

// NewSpaceCmd creates a new command
func NewSpaceCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &SpaceCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("share space", `
Shares a space with another loft user or team. The user
or team need to have access to the cluster.

Example:
loft share space myspace
loft share space myspace --project myproject
loft share space myspace --project myproject --user admin
########################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
########################################################
################# devspace share space #################
########################################################
Shares a space with another loft user or team. The user
or team need to have access to the cluster.

Example:
devspace share space myspace
devspace share space myspace --project myproject
devspace share space myspace --project myproject --user admin
########################################################
	`
	}
	c := &cobra.Command{
		Use:   "space" + util.SpaceNameOnlyUseLine,
		Short: product.Replace("Shares a space with another loft user or team"),
		Long:  description,
		Args:  util.SpaceNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd, args)
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVar(&cmd.Cluster, "cluster", "", "The cluster to use")
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project to use")
	c.Flags().StringVar(&cmd.ClusterRole, "cluster-role", "loft-cluster-space-admin", "The cluster role which is assigned to the user or team for that space")
	c.Flags().StringVar(&cmd.User, "user", "", "The user to share the space with. The user needs to have access to the cluster")
	c.Flags().StringVar(&cmd.Team, "team", "", "The team to share the space with. The team needs to have access to the cluster")
	return c
}

// Run executes the command
func (cmd *SpaceCmd) Run(cobraCmd *cobra.Command, args []string) error {
	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	spaceName := ""
	if len(args) > 0 {
		spaceName = args[0]
	}

	cmd.Cluster, cmd.Project, spaceName, err = helper.SelectSpaceInstanceOrSpace(baseClient, spaceName, cmd.Project, cmd.Cluster, cmd.Log)
	if err != nil {
		return err
	}

	ctx := cobraCmd.Context()

	if cmd.Project == "" {
		return cmd.legacyShareSpace(ctx, baseClient, spaceName)
	}

	return cmd.shareSpace(ctx, baseClient, spaceName)
}

func (cmd *SpaceCmd) shareSpace(ctx context.Context, baseClient client.Client, spaceName string) error {
	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	spaceInstance, err := managementClient.Loft().ManagementV1().SpaceInstances(naming.ProjectNamespace(cmd.Project)).Get(ctx, spaceName, metav1.GetOptions{})
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
	spaceInstance.Spec.ExtraAccessRules = append(spaceInstance.Spec.ExtraAccessRules, accessRule)
	if spaceInstance.Spec.TemplateRef != nil {
		spaceInstance.Spec.TemplateRef.SyncOnce = true
	}
	_, err = managementClient.Loft().ManagementV1().SpaceInstances(naming.ProjectNamespace(cmd.Project)).Update(ctx, spaceInstance, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	if cmd.User != "" {
		cmd.Log.Donef("Successfully granted user %s access to space %s", ansi.Color(cmd.User, "white+b"), ansi.Color(spaceName, "white+b"))
		cmd.Log.Infof("The user can access the space now via: %s", ansi.Color(fmt.Sprintf(product.Replace("loft use space %s --project %s"), spaceName, cmd.Project), "white+b"))
	} else {
		cmd.Log.Donef("Successfully granted team %s access to space %s", ansi.Color(cmd.Team, "white+b"), ansi.Color(spaceName, "white+b"))
		cmd.Log.Infof("The team can access the space now via: %s", ansi.Color(fmt.Sprintf(product.Replace("loft use space %s --project %s"), spaceName, cmd.Project), "white+b"))
	}

	return nil
}

func (cmd *SpaceCmd) legacyShareSpace(ctx context.Context, baseClient client.Client, spaceName string) error {
	userOrTeam, err := createRoleBinding(ctx, baseClient, cmd.Cluster, spaceName, cmd.User, cmd.Team, cmd.ClusterRole, cmd.Log)
	if err != nil {
		return err
	}

	if !userOrTeam.Team {
		cmd.Log.Donef("Successfully granted user %s access to space %s", ansi.Color(userOrTeam.ClusterMember.Info.Name, "white+b"), ansi.Color(spaceName, "white+b"))
		cmd.Log.Infof("The user can access the space now via: %s", ansi.Color(fmt.Sprintf(product.Replace("loft use space %s --cluster %s"), spaceName, cmd.Cluster), "white+b"))
	} else {
		cmd.Log.Donef("Successfully granted team %s access to space %s", ansi.Color(userOrTeam.ClusterMember.Info.Name, "white+b"), ansi.Color(spaceName, "white+b"))
		cmd.Log.Infof("The team can access the space now via: %s", ansi.Color(fmt.Sprintf(product.Replace("loft use space %s --cluster %s"), spaceName, cmd.Cluster), "white+b"))
	}

	return nil
}

func createRoleBinding(ctx context.Context, baseClient client.Client, clusterName, spaceName, userName, teamName, clusterRole string, log log.Logger) (*helper.ClusterUserOrTeam, error) {
	userOrTeam, err := helper.SelectClusterUserOrTeam(baseClient, clusterName, userName, teamName, log)
	if err != nil {
		return nil, err
	}

	clusterClient, err := baseClient.Cluster(clusterName)
	if err != nil {
		return nil, err
	}

	// check if there is already a role binding for this user or team already
	roleBindings, err := clusterClient.RbacV1().RoleBindings(spaceName).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	subjectString := ""
	if userOrTeam.Team {
		subjectString = "loft:team:" + userOrTeam.ClusterMember.Info.Name
	} else {
		subjectString = "loft:user:" + userOrTeam.ClusterMember.Info.Name
	}

	// check if there is already a role binding
	for _, roleBinding := range roleBindings.Items {
		if roleBinding.RoleRef.Kind == "ClusterRole" && roleBinding.RoleRef.Name == clusterRole {
			for _, subject := range roleBinding.Subjects {
				if subject.Kind == "Group" || subject.Kind == "User" {
					if subject.Name == subjectString {
						return nil, nil
					}
				}
			}
		}
	}

	roleBindingName := "loft-user-" + userOrTeam.ClusterMember.Info.Name
	if userOrTeam.Team {
		roleBindingName = "loft-team-" + userOrTeam.ClusterMember.Info.Name
	}
	if len(roleBindingName) > 52 {
		roleBindingName = roleBindingName[:52]
	}

	// create the rolebinding
	_, err = clusterClient.RbacV1().RoleBindings(spaceName).Create(ctx, &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: roleBindingName + "-",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.SchemeGroupVersion.Group,
			Kind:     "ClusterRole",
			Name:     clusterRole,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     "Group",
				APIGroup: rbacv1.GroupName,
				Name:     subjectString,
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "create rolebinding")
	}

	return userOrTeam, nil
}
