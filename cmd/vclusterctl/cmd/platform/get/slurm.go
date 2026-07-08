package get

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	logtable "github.com/loft-sh/log/table"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/platform"
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/loft-sh/vcluster/pkg/platform/slurm"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/duration"
	"sigs.k8s.io/yaml"
)

// slurmCmd holds the get slurm cmd flags.
type slurmCmd struct {
	*flags.GlobalFlags

	Project string
	output  string
	log     log.Logger
}

// newSlurmCmd creates the `get slurm` command.
func newSlurmCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &slurmCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("get slurm", `
Prints details about a Slurm instance, including its conditions and login peer
readiness.

Example:
vcluster platform get slurm my-slurm
vcluster platform get slurm my-slurm --project my-project -o json
########################################################
	`)
	useLine, validator := util.NamedPositionalArgsValidator(true, true, "SLURM_INSTANCE_NAME")
	c := &cobra.Command{
		Use:   "slurm" + useLine,
		Short: "Prints details about a Slurm instance",
		Long:  description,
		Args:  validator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args[0])
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project the Slurm instance belongs to")
	c.Flags().StringVarP(&cmd.output, "output", "o", "text", "Output format. One of: text|json|yaml")
	return c
}

// Run executes the get command.
func (cmd *slurmCmd) Run(ctx context.Context, name string) error {
	platformClient, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.log))
	if err != nil {
		return err
	}

	_, project, err := platform.SelectProjectOrCluster(ctx, platformClient, "", cmd.Project, false, cmd.log)
	if err != nil {
		return err
	}

	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	instance, err := slurm.Get(ctx, managementClient, project, name)
	if err != nil {
		return err
	}

	switch cmd.output {
	case OutputJSON:
		out, err := json.MarshalIndent(instance, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(out))
		return nil
	case OutputYAML:
		out, err := yaml.Marshal(instance)
		if err != nil {
			return err
		}
		fmt.Fprint(os.Stdout, string(out))
		return nil
	default:
		return cmd.printText(project, instance)
	}
}

func (cmd *slurmCmd) printText(project string, instance *managementv1.SlurmInstance) error {
	tenant := "-"
	if instance.Status.VirtualClusterInstance != nil {
		tenant = instance.Status.VirtualClusterInstance.Name
		if instance.Status.VirtualClusterInstance.Cluster != "" {
			tenant = fmt.Sprintf("%s (cluster %s)", tenant, instance.Status.VirtualClusterInstance.Cluster)
		}
	}

	cmd.log.Infof("Name:           %s", instance.Name)
	cmd.log.Infof("Project:        %s", project)
	cmd.log.Infof("Phase:          %s", orDash(string(instance.Status.Phase)))
	cmd.log.Infof("Template:       %s", orDash(instance.Spec.VirtualCluster.Template.Name))
	cmd.log.Infof("Tenant Cluster: %s", tenant)
	cmd.log.Infof("Age:            %s", duration.HumanDuration(time.Since(instance.CreationTimestamp.Time)))
	if instance.Status.Message != "" {
		cmd.log.Infof("Message:        %s", instance.Status.Message)
	}

	if len(instance.Status.Conditions) == 0 {
		return nil
	}

	header := []string{"Condition", "Status", "Reason", "Message"}
	values := [][]string{}
	for _, c := range instance.Status.Conditions {
		values = append(values, []string{
			string(c.Type),
			string(c.Status),
			c.Reason,
			c.Message,
		})
	}
	logtable.PrintTable(cmd.log, header, values)
	return nil
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
