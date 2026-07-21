package wakeup

import (
	"context"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/spf13/cobra"
)

// VClusterCmd holds the login cmd flags
type VClusterCmd struct {
	*flags.GlobalFlags
	cli.ResumeOptions

	log log.Logger
}

// NewVClusterCmd creates a new command
func NewVClusterCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &VClusterCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "vcluster" + util.VClusterNameOnlyUseLine,
		Short: "Lists all virtual clusters that are connected to the current platform",
		Long: `#########################################################################
################### vcluster platform wakeup vcluster ###################
#########################################################################
Wakeup will start a virtual cluster after it was put to sleep.
vCluster will recreate all the workloads after it has
started automatically.

Example:
vcluster platform wakeup vcluster test --namespace test
#########################################################################
	`,
		Args: util.VClusterNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	// Platform flags
	cobraCmd.Flags().StringVar(&cmd.Project, "project", "", "The vCluster platform project to use")

	return cobraCmd
}

func (cmd *VClusterCmd) Run(ctx context.Context, args []string) error {
	return cli.ResumePlatform(ctx, &cmd.ResumeOptions, cmd.LoadedConfig(cmd.log), args[0], cmd.log)
}
