package create

import (
	"context"
	"fmt"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/platform"
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/loft-sh/vcluster/pkg/platform/slurm"
	"github.com/loft-sh/vcluster/pkg/projectutil"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SlurmCmd holds the create slurm cmd flags.
type SlurmCmd struct {
	*flags.GlobalFlags

	Project         string
	Template        string
	TemplateVersion string
	Parameters      string
	SetParams       []string
	Wait            bool

	log log.Logger
}

// newSlurmCmd creates the `create slurm` command.
func newSlurmCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &SlurmCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("create slurm", `
Creates a new Slurm instance from a tenant cluster template.

Example:
vcluster platform create slurm my-slurm --tenant-cluster-template slurm-host
vcluster platform create slurm my-slurm --tenant-cluster-template slurm-host --set nodes=4
vcluster platform create slurm my-slurm --tenant-cluster-template slurm-host --parameters params.yaml
########################################################
	`)
	useLine, validator := util.NamedPositionalArgsValidator(true, true, "SLURM_INSTANCE_NAME")
	c := &cobra.Command{
		Use:   "slurm" + useLine,
		Short: "Creates a new Slurm instance",
		Long:  description,
		Args:  validator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args[0])
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project to create the Slurm instance in")
	c.Flags().StringVar(&cmd.Template, "tenant-cluster-template", "", "The tenant cluster template to provision the Slurm instance from")
	c.Flags().StringVar(&cmd.TemplateVersion, "tenant-cluster-template-version", "", "The tenant cluster template version to use (defaults to the latest)")
	c.Flags().StringVar(&cmd.Parameters, "parameters", "", "A file with YAML template parameters")
	c.Flags().StringArrayVar(&cmd.SetParams, "set", []string{}, "Set template parameters. E.g. --set nodes=4")
	c.Flags().BoolVar(&cmd.Wait, "wait", false, "Wait until the Slurm instance is ready for SSH access")
	return c
}

// Run executes the create command.
func (cmd *SlurmCmd) Run(ctx context.Context, name string) error {
	if cmd.Template == "" {
		return fmt.Errorf("please specify a tenant cluster template with --tenant-cluster-template")
	}

	platformClient, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.log))
	if err != nil {
		return err
	}

	_, project, err := platform.SelectProjectOrCluster(ctx, platformClient, "", cmd.Project, false, cmd.log)
	if err != nil {
		return err
	}

	template, resolvedParameters, err := platform.ResolveVirtualClusterTemplate(ctx, platformClient, project, cmd.Template, cmd.TemplateVersion, cmd.SetParams, cmd.Parameters, cmd.log)
	if err != nil {
		return err
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	instance := &managementv1.SlurmInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: projectutil.ProjectNamespace(project),
		},
		Spec: managementv1.SlurmInstanceSpec{
			SlurmInstanceSpec: storagev1.SlurmInstanceSpec{
				VirtualCluster: storagev1.SlurmVirtualCluster{
					Template: storagev1.SlurmVirtualClusterTemplate{
						Name:       template.Name,
						Version:    cmd.TemplateVersion,
						Parameters: resolvedParameters,
					},
				},
			},
		},
	}

	_, err = managementClient.Loft().ManagementV1().SlurmInstances(instance.Namespace).Create(ctx, instance, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create slurm instance: %w", err)
	}
	cmd.log.Donef("Successfully created Slurm instance %s in project %s", ansi.Color(name, "white+b"), ansi.Color(project, "white+b"))

	if cmd.Wait {
		if _, err := slurm.WaitForTailnetReady(ctx, managementClient, project, name, true, cmd.log); err != nil {
			return err
		}
		cmd.log.Donef("Slurm instance %s is ready for SSH access", ansi.Color(name, "white+b"))
	}

	return nil
}
