package cmd

import (
	"context"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/spf13/cobra"
)

// ResumeCmd holds the cmd flags
type ResumeCmd struct {
	*flags.GlobalFlags
	cli.ResumeOptions

	Log log.Logger
}

// NewResumeCmd creates a new command
func NewResumeCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ResumeCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:     "resume" + util.VClusterNameOnlyUseLine,
		Aliases: []string{"wakeup"},
		Short:   "Resumes a virtual cluster",
		Long: `#######################################################
################### vcluster resume ###################
#######################################################
Resume will start a vcluster after it was paused.
vcluster will recreate all the workloads after it has
started automatically.

Example:
vcluster resume test --namespace test
#######################################################
	`,
		Args:              util.VClusterNameOnlyValidator,
		ValidArgsFunction: newValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Manager, "manager", "", "The manager to use for managing the virtual cluster, can be either helm or platform.")

	// Platform flags
	cobraCmd.Flags().StringVar(&cmd.Project, "project", "", "[PLATFORM] The vCluster platform project to use")

	return cobraCmd
}

// Run executes the functionality
func (cmd *ResumeCmd) Run(ctx context.Context, args []string) error {
	cfg := cmd.LoadedConfig(cmd.Log)
	// check if we should resume a platform backed virtual cluster
	if cfg.Manager.Type == config.ManagerPlatform {
		return cli.ResumePlatform(ctx, &cmd.ResumeOptions, cfg, args[0], cmd.Log)
	}

	return cli.ResumeHelm(ctx, cmd.GlobalFlags, args[0], cmd.Log)
}
