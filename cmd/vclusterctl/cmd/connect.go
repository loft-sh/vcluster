package cmd

import (
	"cmp"
	"context"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/flags/connect"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/spf13/cobra"
)

// ConnectCmd holds the cmd flags
type ConnectCmd struct {
	Log log.Logger
	*flags.GlobalFlags
	cli.ConnectOptions
}

// NewConnectCmd creates a new command
func NewConnectCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &ConnectCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	useLine, nameValidator := util.NamedPositionalArgsValidator(true, false, "VCLUSTER_NAME")

	cobraCmd := &cobra.Command{
		Use:   "connect" + useLine,
		Short: "Connect to a virtual cluster",
		Long: `#######################################################
################## vcluster connect ###################
#######################################################
Connect to a virtual cluster

Example:
vcluster connect test --namespace test
# Open a new bash with the vcluster KUBECONFIG defined
vcluster connect test -n test -- bash
vcluster connect test -n test -- kubectl get ns
#######################################################
	`,
		Args:              nameValidator,
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Driver, "driver", "", "The driver to use for managing the virtual cluster, can be either helm or platform.")

	connect.AddCommonFlags(cobraCmd, &cmd.ConnectOptions)
	connect.AddPlatformFlags(cobraCmd, &cmd.ConnectOptions, "[PLATFORM] ")

	return cobraCmd
}

// Run executes the functionality
func (cmd *ConnectCmd) Run(ctx context.Context, args []string) error {
	vClusterName := ""
	if len(args) > 0 {
		vClusterName = args[0]
	}

	// validate flags
	err := cmd.validateFlags()
	if err != nil {
		return err
	}

	cfg := cmd.LoadedConfig(cmd.Log)

	// If driver has been passed as flag use it, otherwise read it from the config file
	driverType, err := config.ParseDriverType(cmp.Or(cmd.Driver, string(cfg.Driver.Type)))
	if err != nil {
		return fmt.Errorf("parse driver type: %w", err)
	}

	if driverType == config.PlatformDriver {
		return cli.ConnectPlatform(ctx, &cmd.ConnectOptions, cmd.GlobalFlags, vClusterName, args[1:], cmd.Log)
	}

	return cli.ConnectHelm(ctx, &cmd.ConnectOptions, cmd.GlobalFlags, vClusterName, args[1:], cmd.Log)
}

func (cmd *ConnectCmd) validateFlags() error {
	if cmd.ServiceAccountClusterRole != "" && cmd.ServiceAccount == "" {
		return fmt.Errorf("expected --service-account to be defined as well")
	}

	return nil
}
