package deletecmd

import (
	"context"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	flagsdelete "github.com/loft-sh/vcluster/pkg/cli/flags/delete"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/spf13/cobra"
)

// VClusterCmd holds the cmd flags
type VClusterCmd struct {
	*flags.GlobalFlags
	cli.DeleteOptions

	log log.Logger
}

// newVClusterCmd creates a new command
func newVClusterCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &VClusterCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "vcluster" + util.VClusterNameOnlyUseLine,
		Short: "Deletes a virtual cluster",
		Long: `#########################################################################
################### vcluster platform delete vcluster ###################
#########################################################################
Deletes a virtual cluster

Example:
vcluster platform delete vcluster --namespace test
#########################################################################
	`,
		Args:              util.VClusterNameOnlyValidator,
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	flagsdelete.AddCommonFlags(cobraCmd, &cmd.DeleteOptions)
	flagsdelete.AddPlatformFlags(cobraCmd, &cmd.DeleteOptions)

	return cobraCmd
}

// Run executes the functionality
func (cmd *VClusterCmd) Run(ctx context.Context, args []string) error {
	return cli.DeletePlatform(ctx, &cmd.DeleteOptions, cmd.LoadedConfig(cmd.log), args[0], cmd.log)
}
