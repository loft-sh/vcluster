package deletecmd

import (
	"context"
	"fmt"
	"time"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/loft-sh/vcluster/pkg/platform/kube"
	"github.com/loft-sh/vcluster/pkg/projectutil"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// SlurmCmd holds the delete slurm cmd flags.
type SlurmCmd struct {
	*flags.GlobalFlags

	Project string
	Wait    bool

	log log.Logger
}

// newSlurmCmd creates the `delete slurm` command.
func newSlurmCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &SlurmCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("delete slurm", `
Deletes a Slurm instance. The controller tears down the tenant cluster, the
login access key and any tailnet peers.

Example:
vcluster platform delete slurm my-slurm
vcluster platform delete slurm my-slurm --project my-project --wait
########################################################
	`)
	useLine, validator := util.NamedPositionalArgsValidator(true, true, "SLURM_INSTANCE_NAME")
	c := &cobra.Command{
		Use:   "slurm" + useLine,
		Short: "Deletes a Slurm instance",
		Long:  description,
		Args:  validator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args[0])
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project the Slurm instance belongs to")
	c.Flags().BoolVar(&cmd.Wait, "wait", false, "Wait until the Slurm instance is fully removed")
	return c
}

// Run executes the delete command.
func (cmd *SlurmCmd) Run(ctx context.Context, name string) error {
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

	namespace := projectutil.ProjectNamespace(project)
	if err := managementClient.Loft().ManagementV1().SlurmInstances(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("delete slurm instance: %w", err)
	}
	cmd.log.Donef("Successfully deleted Slurm instance %s in project %s", ansi.Color(name, "white+b"), ansi.Color(project, "white+b"))

	if cmd.Wait {
		return cmd.waitForRemoval(ctx, managementClient, namespace, name)
	}
	return nil
}

func (cmd *SlurmCmd) waitForRemoval(ctx context.Context, managementClient kube.Interface, namespace, name string) error {
	cmd.log.Infof("Waiting for Slurm instance %s to be fully removed...", name)
	return wait.PollUntilContextTimeout(ctx, time.Second, clihelper.Timeout(), true, func(ctx context.Context) (bool, error) {
		_, err := managementClient.Loft().ManagementV1().SlurmInstances(namespace).Get(ctx, name, metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		return false, nil
	})
}
