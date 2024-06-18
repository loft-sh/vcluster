package wakeup

import (
	"context"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/platform"
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/loft-sh/vcluster/pkg/projectutil"
	"github.com/spf13/cobra"
)

// NamespaceCmd holds the cmd flags
type NamespaceCmd struct {
	*flags.GlobalFlags

	Project string
	Cluster string
	Log     log.Logger
}

// NewNamespaceCmd creates a new command
func NewNamespaceCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
	cmd := &NamespaceCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("wakeup namespace", `
wakeup resumes a sleeping vCluster platform namespace
Example:
vcluster platform wakeup namespace myspace
vcluster platform wakeup namespace myspace --project myproject
#######################################################
	`)
	c := &cobra.Command{
		Use:     "namespace" + util.NamespaceNameOnlyUseLine,
		Short:   "Wakes up a vCluster platform namespace",
		Long:    description,
		Args:    util.NamespaceNameOnlyValidator,
		Aliases: []string{"space"},
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	c.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project to use")
	c.Flags().StringVar(&cmd.Cluster, "cluster", "", "The cluster to use")
	return c
}

// Run executes the functionality
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

	return cmd.spaceWakeUp(ctx, platformClient, spaceName)
}

func (cmd *NamespaceCmd) spaceWakeUp(ctx context.Context, platformClient platform.Client, spaceName string) error {
	managementClient, err := platformClient.Management()
	if err != nil {
		return err
	}

	_, err = platform.WaitForSpaceInstance(ctx, managementClient, projectutil.ProjectNamespace(cmd.Project), spaceName, true, cmd.Log)
	if err != nil {
		return err
	}

	return nil
}
