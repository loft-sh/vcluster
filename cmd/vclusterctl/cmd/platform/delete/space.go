package deletecmd

import (
	"context"
	"time"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/platform"
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/loft-sh/vcluster/pkg/platform/kube"
	"github.com/loft-sh/vcluster/pkg/platform/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/projectutil"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/mgutz/ansi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NamespaceCmd holds the cmd flags
type NamespaceCmd struct {
	*flags.GlobalFlags

	Cluster       string
	Project       string
	DeleteContext bool
	Wait          bool

	Log log.Logger
}

// newNamespaceCmd creates a new command
func newNamespaceCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &NamespaceCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("delete namespace", `
Deletes a vCluster platform namespace from a cluster

Example:
vcluster platform delete namespace myspace
vcluster platform delete namespace myspace --project myproject
########################################################
	`)
	c := &cobra.Command{
		Use:     "namespace" + util.NamespaceNameOnlyUseLine,
		Short:   "Deletes a vCluster platform namespace from a cluster",
		Long:    description,
		Args:    util.NamespaceNameOnlyValidator,
		Aliases: []string{"space"},
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVar(&cmd.Cluster, "cluster", "", "The cluster to use")
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project to use")
	c.Flags().BoolVar(&cmd.DeleteContext, "delete-context", true, "If the corresponding kube context should be deleted if there is any")
	c.Flags().BoolVar(&cmd.Wait, "wait", false, "Termination of this command waits for namespace to be deleted")
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

	return cmd.deleteSpace(ctx, platformClient, spaceName)
}

func (cmd *NamespaceCmd) deleteSpace(ctx context.Context, platformClient platform.Client, spaceName string) error {
	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	err = managementClient.Loft().ManagementV1().SpaceInstances(projectutil.ProjectNamespace(cmd.Project)).Delete(ctx, spaceName, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "delete namespace")
	}

	cmd.Log.Donef("Successfully deleted namespace %s in project %s", ansi.Color(spaceName, "white+b"), ansi.Color(cmd.Project, "white+b"))

	// update kube config
	if cmd.DeleteContext {
		err = kubeconfig.DeleteContext(kubeconfig.SpaceInstanceContextName(cmd.Project, spaceName))
		if err != nil {
			return err
		}

		cmd.Log.Donef("Successfully deleted kube context for namespace %s", ansi.Color(spaceName, "white+b"))
	}

	// wait until deleted
	if cmd.Wait {
		cmd.Log.Info("Waiting for namespace to be deleted...")
		for isSpaceInstanceStillThere(ctx, managementClient, projectutil.ProjectNamespace(cmd.Project), spaceName) {
			time.Sleep(time.Second)
		}
		cmd.Log.Done("Namespace is deleted")
	}

	return nil
}

func isSpaceInstanceStillThere(ctx context.Context, managementClient kube.Interface, spaceInstanceNamespace, spaceName string) bool {
	_, err := managementClient.Loft().ManagementV1().SpaceInstances(spaceInstanceNamespace).Get(ctx, spaceName, metav1.GetOptions{})
	return err == nil
}
