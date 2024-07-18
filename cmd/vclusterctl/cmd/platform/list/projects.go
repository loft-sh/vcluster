package list

import (
	"context"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectsCmd holds the login cmd flags
type ProjectsCmd struct {
	*flags.GlobalFlags

	log log.Logger
}

// newProjectsCmd creates a new spaces command
func newProjectsCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ProjectsCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("list projects", `
List the vcluster platform projects you have access to

Example:
vcluster platform list projects
########################################################
	`)
	projectsCmd := &cobra.Command{
		Use:   "projects",
		Short: product.Replace("Lists the loft projects you have access to"),
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.RunProjects(cobraCmd.Context())
		},
	}

	return projectsCmd
}

// RunProjects executes the functionality
func (cmd *ProjectsCmd) RunProjects(ctx context.Context) error {
	platformClient, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.log))
	if err != nil {
		return err
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	projectList, err := managementClient.Loft().ManagementV1().Projects().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	header := []string{
		"Project",
	}
	projects := make([][]string, len(projectList.Items))
	for i, project := range projectList.Items {
		projects[i] = []string{project.Name}
	}

	table.PrintTable(cmd.log, header, projects)
	return nil
}
