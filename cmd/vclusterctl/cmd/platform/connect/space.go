package connect

import (
	"context"
	"os"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/platform"
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/loft-sh/vcluster/pkg/platform/kubeconfig"
	"github.com/loft-sh/vcluster/pkg/projectutil"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

// NamespaceCmd holds the cmd flags
type NamespaceCmd struct {
	*flags.GlobalFlags

	Cluster                      string
	Project                      string
	Print                        bool
	SkipWait                     bool
	DisableDirectClusterEndpoint bool

	log log.Logger
}

// newNamespaceCmd creates a new command
func newNamespaceCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &NamespaceCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("connect namespace", `
Creates a new kube context for the given vCluster platform namespace.

Example:
vcluster platform connect namespace
vcluster platform connect namespace myspace
vcluster platform connect namespace myspace --project myproject
########################################################
	`)
	useLine, validator := util.NamedPositionalArgsValidator(false, false, "SPACE_NAME")
	c := &cobra.Command{
		Use:     "namespace" + useLine,
		Short:   "Creates a kube context for the given vCluster platform namespace",
		Long:    description,
		Args:    validator,
		Aliases: []string{"space"},
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			if !cmd.Print {
				upgrade.PrintNewerVersionWarning()
			}

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVar(&cmd.Cluster, "cluster", "", "The cluster to use")
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project to use")
	c.Flags().BoolVar(&cmd.Print, "print", false, "When enabled prints the context to stdout")
	c.Flags().BoolVar(&cmd.SkipWait, "skip-wait", false, "If true, will not wait until the namespace is running")
	c.Flags().BoolVar(&cmd.DisableDirectClusterEndpoint, "disable-direct-cluster-endpoint", false, "When enabled does not use an available direct cluster endpoint to connect to the cluster")
	return c
}

// Run executes the command
func (cmd *NamespaceCmd) Run(ctx context.Context, args []string) error {
	cfg := cmd.LoadedConfig(cmd.log)
	platformClient, err := platform.InitClientFromConfig(ctx, cfg)
	if err != nil {
		return err
	}

	spaceName := ""
	if len(args) > 0 {
		spaceName = args[0]
	}

	cmd.Cluster, cmd.Project, spaceName, err = platform.SelectSpaceInstance(ctx, platformClient, spaceName, cmd.Project, cmd.log)
	if err != nil {
		return err
	}

	return cmd.connectSpace(ctx, platformClient, spaceName, cfg)
}

func (cmd *NamespaceCmd) connectSpace(ctx context.Context, platformClient platform.Client, spaceName string, cfg *config.CLI) error {
	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	// wait until space is ready
	spaceInstance, err := platform.WaitForSpaceInstance(ctx, managementClient, projectutil.ProjectNamespace(cmd.Project), spaceName, !cmd.SkipWait, cmd.log)
	if err != nil {
		return err
	}

	// create kube context options
	contextOptions, err := platform.CreateSpaceInstanceOptions(ctx, platformClient, cmd.Config, cmd.Project, spaceInstance, true)
	if err != nil {
		return err
	}

	// check if we should print or update the config
	if cmd.Print {
		err = kubeconfig.PrintKubeConfigTo(contextOptions, os.Stdout)
		if err != nil {
			return err
		}
	} else {
		// update kube config
		err = kubeconfig.UpdateKubeConfig(contextOptions, cfg)
		if err != nil {
			return err
		}

		cmd.log.Donef("Successfully updated kube context to use namespace %s in project %s", ansi.Color(spaceName, "white+b"), ansi.Color(cmd.Project, "white+b"))
	}

	return nil
}
