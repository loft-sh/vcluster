package list

import (
	"context"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectsCmd holds the login cmd flags
type ProjectsCmd struct {
	*flags.GlobalFlags
	cli.ListOptions
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

	AddCommonFlags(projectsCmd, &cmd.ListOptions)
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

	// Define a function to extract specific fields from a Project struct.
	// This function will be passed to PrintData to determine which fields
	// should be printed in JSON or table format.
	getValuesFunc := func(p managementv1.Project) []string {
		return []string{p.Name}
	}

	err = PrintData(cmd.log, cmd.Output, header, projectList.Items, getValuesFunc)
	return err
}
