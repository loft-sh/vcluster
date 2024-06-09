package connect

import (
	"context"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/flags/connect"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/spf13/cobra"
)

// VClusterCmd holds the cmd flags
type VClusterCmd struct {
	Log log.Logger
	*flags.GlobalFlags
	cli.ConnectOptions
}

// newVClusterCmd creates a new command
func newVClusterCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &VClusterCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	useLine, nameValidator := util.NamedPositionalArgsValidator(true, false, "VCLUSTER_NAME")

	cobraCmd := &cobra.Command{
		Use:   "vcluster" + useLine,
		Short: "Connect to a virtual cluster",
		Long: `#########################################################################
################## vcluster platform connect vcluster ###################
#########################################################################
Connect to a virtual cluster

Example:
vcluster platform connect vcluster test --namespace test
# Open a new bash with the vcluster KUBECONFIG defined
vcluster platform connect vcluster test -n test -- bash
vcluster platform connect vcluster test -n test -- kubectl get ns
#########################################################################
	`,
		Args:              nameValidator,
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	connect.AddCommonFlags(cobraCmd, &cmd.ConnectOptions)
	connect.AddPlatformFlags(cobraCmd, &cmd.ConnectOptions)

	return cobraCmd
}

// Run executes the functionality
func (cmd *VClusterCmd) Run(ctx context.Context, args []string) error {
	vClusterName := ""
	if len(args) > 0 {
		vClusterName = args[0]
	}

	// validate flags
	err := cmd.validateFlags()
	if err != nil {
		return err
	}

	return cli.ConnectPlatform(ctx, &cmd.ConnectOptions, cmd.GlobalFlags, vClusterName, args[1:], cmd.Log)
}

func (cmd *VClusterCmd) validateFlags() error {
	if cmd.ServiceAccountClusterRole != "" && cmd.ServiceAccount == "" {
		return fmt.Errorf("expected --service-account to be defined as well")
	}

	return nil
}
