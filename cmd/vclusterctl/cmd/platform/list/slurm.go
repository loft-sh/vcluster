package list

import (
	"context"
	"time"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/loft-sh/vcluster/pkg/platform/kube"
	"github.com/loft-sh/vcluster/pkg/platform/slurm"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
)

// SlurmCmd holds the list slurm cmd flags.
type SlurmCmd struct {
	*flags.GlobalFlags
	cli.ListOptions

	Project string
	log     log.Logger
}

// slurmRow pairs a SlurmInstance with the project it was found in for table
// rendering.
type slurmRow struct {
	project  string
	instance managementv1.SlurmInstance
}

// newSlurmCmd creates the `list slurm` command.
func newSlurmCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &SlurmCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("list slurm", `
List the Slurm instances you have access to.

Example:
vcluster platform list slurm
vcluster platform list slurm --project my-project
########################################################
	`)
	c := &cobra.Command{
		Use:   "slurm",
		Short: "Lists the Slurm instances you have access to",
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project to list Slurm instances from (defaults to all accessible projects)")
	AddCommonFlags(c, &cmd.ListOptions)
	return c
}

// Run executes the list command.
func (cmd *SlurmCmd) Run(ctx context.Context) error {
	platformClient, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.log))
	if err != nil {
		return err
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	projects, err := cmd.resolveProjects(ctx, managementClient)
	if err != nil {
		return err
	}

	rows := []slurmRow{}
	for _, project := range projects {
		instanceList, err := slurm.List(ctx, managementClient, project)
		if err != nil {
			// Skip projects the user cannot list Slurm instances in.
			cmd.log.Debugf("skip project %s: %v", project, err)
			continue
		}
		for i := range instanceList.Items {
			rows = append(rows, slurmRow{project: project, instance: instanceList.Items[i]})
		}
	}

	header := []string{"Name", "Project", "Phase", "Tenant Cluster", "Age"}
	getValues := func(r slurmRow) []string {
		return []string{
			clihelper.GetTableDisplayName(r.instance.Name, r.instance.Spec.DisplayName),
			r.project,
			string(r.instance.Status.Phase),
			tenantClusterName(&r.instance),
			duration.HumanDuration(time.Since(r.instance.CreationTimestamp.Time)),
		}
	}

	return PrintData(cmd.log, cmd.Output, header, rows, getValues)
}

// resolveProjects returns the single requested project, or all accessible ones.
func (cmd *SlurmCmd) resolveProjects(ctx context.Context, managementClient kube.Interface) ([]string, error) {
	if cmd.Project != "" {
		return []string{cmd.Project}, nil
	}

	projectList, err := managementClient.Loft().ManagementV1().Projects().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	projects := make([]string, 0, len(projectList.Items))
	for _, project := range projectList.Items {
		projects = append(projects, project.Name)
	}
	return projects, nil
}

func tenantClusterName(instance *managementv1.SlurmInstance) string {
	if instance.Status.VirtualClusterInstance != nil {
		return instance.Status.VirtualClusterInstance.Name
	}
	return ""
}
