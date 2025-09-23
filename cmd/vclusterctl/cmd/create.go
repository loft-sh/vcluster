package cmd

import (
	"cmp"
	"errors"
	"fmt"
	"strings"

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

	log log.Logger
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

			return cmd.Run(cobraCmd, newArgs)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Driver, "driver", "", "The driver to use for managing the virtual cluster, can be either helm or platform.")

	create.AddCommonFlags(cobraCmd, &cmd.CreateOptions)
	create.AddHelmFlags(cobraCmd, &cmd.CreateOptions)
	create.AddPlatformFlags(cobraCmd, &cmd.CreateOptions, "[PLATFORM] ")

	return cobraCmd
}

// Run executes the functionality
func (cmd *CreateCmd) Run(cobraCmd *cobra.Command, args []string) error {
	if !cmd.UpdateCurrent {
		cmd.log.Warnf("%q has no effect anymore. Please consider using %q", "--update-current=false", "--connect=false")
	}

	cfg := cmd.LoadedConfig(cmd.log)

	// If driver has been passed as flag use it, otherwise read it from the config file
	driver, err := config.ParseDriverType(cmp.Or(cmd.Driver, string(cfg.Driver.Type)))
	if err != nil {
		return fmt.Errorf("parse driver type: %w", err)
	}

	ctx := cobraCmd.Context()

	// check if there is a platform client or we skip the info message
	_, err = platform.InitClientFromConfig(ctx, cfg)
	if err == nil {
		config.PrintDriverInfo("create", driver, cmd.log)
	}

	// check if we should create a platform vCluster
	if driver == config.PlatformDriver {
		return cli.CreatePlatform(ctx, &cmd.CreateOptions, cmd.GlobalFlags, args[0], cmd.log)
	}

	// log error if platform flags have been set when using driver helm
	var fs []string
	pfs := create.ChangedPlatformFlags(cobraCmd)
	for pf, changed := range pfs {
		if changed {
			fs = append(fs, pf)
		}
	}

	if len(fs) > 0 {
		cmd.log.Fatalf("Following platform flags have been set, which won't have any effect when using driver type %s: %s", config.HelmDriver, strings.Join(fs, ", "))
	}

	return cli.CreateHelm(ctx, &cmd.CreateOptions, cmd.GlobalFlags, args[0], cmd.log)
}
