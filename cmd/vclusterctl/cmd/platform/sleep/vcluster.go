package sleep

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
	cli.PauseOptions

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
		Short: "Put a virtual cluster to sleep",
		Long: `########################################################################
################### vcluster platform sleep vcluster ###################
########################################################################
Sleep will stop a virtual cluster and free all its used
computing resources.

Sleep will scale down the virtual cluster and delete
all workloads created through the virtual cluster. Upon resume,
all workloads will be recreated. Other resources such
as persistent volume claims, services etc. will not be affected.

Example:
vcluster platform sleep vcluster test --namespace test
########################################################################
	`,
		Args: util.VClusterNameOnlyValidator,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	// Platform flags
	cobraCmd.Flags().StringVar(&cmd.Project, "project", "", "The vCluster platform project to use")
	cobraCmd.Flags().Int64Var(&cmd.ForceDuration, "prevent-wakeup", -1, "The amount of seconds this vcluster should sleep until it can be woken up again (use 0 for infinite sleeping). During this time the space can only be woken up by `vcluster resume vcluster`, manually deleting the annotation on the namespace or through the loft UI")

	return cobraCmd
}

func (cmd *VClusterCmd) Run(ctx context.Context, args []string) error {
	return cli.PausePlatform(ctx, &cmd.PauseOptions, cmd.LoadedConfig(cmd.log), args[0], cmd.log)
}
