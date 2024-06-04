package importcmd

import (
	"context"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/spf13/cobra"
)

type VClusterCmd struct {
	*flags.GlobalFlags
	cli.ImportOptions

	Log log.Logger
}

func NewVClusterCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &VClusterCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := `#################################################################
############### vcluster platform import vcluster ###############
#################################################################
Imports a vCluster into a vCluster platform project.

Example:
vcluster platform import vcluster my-vcluster --cluster connected-cluster \
--namespace vcluster-my-vcluster --project my-project --import-name my-vcluster
################################################################
	`

	importCmd := &cobra.Command{
		Use:   "vcluster" + util.VClusterNameOnlyUseLine,
		Short: "Imports a vCluster into a vCluster platform project",
		Long:  description,
		Args:  util.VClusterNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	importCmd.Flags().StringVar(&cmd.ClusterName, "cluster", "", "Cluster name of the cluster the virtual cluster is running on")
	importCmd.Flags().StringVar(&cmd.Project, "project", "", "The project to import the vCluster into")
	importCmd.Flags().StringVar(&cmd.ImportName, "import-name", "", "The name of the vCluster under projects. If unspecified, will use the vcluster name")

	return importCmd
}

// Run executes the functionality
func (cmd *VClusterCmd) Run(ctx context.Context, args []string) error {
	return cli.ImportPlatform(ctx, &cmd.ImportOptions, cmd.GlobalFlags, args[0], cmd.Log)
}
