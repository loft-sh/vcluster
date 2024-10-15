package cmd

import (
	"cmp"
	"context"
	"errors"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/flags/create"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/spf13/cobra"
)

// CreateCmd holds the login cmd flags
type CreateCmd struct {
	*flags.GlobalFlags
	cli.CreateOptions

	log            log.Logger
	reuseNamespace bool
}

// NewCreateCmd creates a new command
func NewCreateCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &CreateCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "create" + util.VClusterNameOnlyUseLine,
		Short: "Create a new virtual cluster",
		Long: `#######################################################
################### vcluster create ###################
#######################################################
Creates a new virtual cluster

Example:
vcluster create test --namespace test
#######################################################
	`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			newArgs, err := util.PromptForArgs(cmd.log, args, "vcluster name")
			if err != nil {
				switch {
				case errors.Is(err, util.ErrNonInteractive):
					if err := util.VClusterNameOnlyValidator(cobraCmd, args); err != nil {
						return err
					}
				default:
					return err
				}
			}

			// Check for newer version
			upgrade.PrintNewerVersionWarning()

			return cmd.Run(cobraCmd.Context(), newArgs)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Driver, "driver", "", "The driver to use for managing the virtual cluster, can be either helm or platform.")
	cobraCmd.Flags().BoolVar(&cmd.reuseNamespace, "reuse-namespace", false, "Allows to create multiple virtual clusters in a single namespace")
	cobraCmd.Flag("reuse-namespace").Hidden = true

	create.AddCommonFlags(cobraCmd, &cmd.CreateOptions)
	create.AddHelmFlags(cobraCmd, &cmd.CreateOptions)
	create.AddPlatformFlags(cobraCmd, &cmd.CreateOptions, "[PLATFORM] ")

	return cobraCmd
}

// Run executes the functionality
func (cmd *CreateCmd) Run(ctx context.Context, args []string) error {
	if !cmd.UpdateCurrent {
		cmd.log.Warnf("%q has no effect anymore. Please consider using %q", "--update-current=false", "--connect=false")
	}

	cfg := cmd.LoadedConfig(cmd.log)

	// If driver has been passed as flag use it, otherwise read it from the config file
	driver, err := config.ParseDriverType(cmp.Or(cmd.Driver, string(cfg.Driver.Type)))
	if err != nil {
		return fmt.Errorf("parse driver type: %w", err)
	}

	// check if there is a platform client or we skip the info message
	_, err = platform.InitClientFromConfig(ctx, cfg)
	if err == nil {
		config.PrintDriverInfo("create", driver, cmd.log)
	}

	// check if we should create a platform vCluster
	if driver == config.PlatformDriver {
		return cli.CreatePlatform(ctx, &cmd.CreateOptions, cmd.GlobalFlags, args[0], cmd.log)
	}

	return cli.CreateHelm(ctx, &cmd.CreateOptions, cmd.GlobalFlags, args[0], cmd.log, cmd.reuseNamespace)
}
