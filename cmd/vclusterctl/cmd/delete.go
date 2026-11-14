package cmd

import (
	"cmp"
	"fmt"
	"strings"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	flagsdelete "github.com/loft-sh/vcluster/pkg/cli/flags/delete"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/spf13/cobra"
)

// DeleteCmd holds the delete cmd flags
type DeleteCmd struct {
	*flags.GlobalFlags
	cli.DeleteOptions

	log log.Logger
}

// NewDeleteCmd creates a new command
func NewDeleteCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &DeleteCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "delete [VCLUSTER_NAME]",
		Short: "Deletes a virtual cluster",
		Long: `#######################################################
################### vcluster delete ###################
#######################################################
Deletes a virtual cluster

Example:
vcluster delete test --namespace test
vcluster delete --all
vcluster delete --all --namespace test
#######################################################
	`,
		Args: func(cobraCmd *cobra.Command, args []string) error {
			return validateDeleteArgs(cobraCmd, args, cmd.All)
		},
		Aliases:           []string{"rm"},
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd, args)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Driver, "driver", "", "The driver to use for managing the virtual cluster, can be either helm, platform, or docker.")

	flagsdelete.AddCommonFlags(cobraCmd, &cmd.DeleteOptions)
	flagsdelete.AddDeleteAllFlags(cobraCmd, &cmd.DeleteOptions)
	flagsdelete.AddHelmFlags(cobraCmd, &cmd.DeleteOptions)
	flagsdelete.AddPlatformFlags(cobraCmd, &cmd.DeleteOptions, "[PLATFORM] ")

	return cobraCmd
}

// validateDeleteArgs requires exactly one VCLUSTER_NAME unless --all is set, in
// which case no name may be given.
func validateDeleteArgs(cobraCmd *cobra.Command, args []string, all bool) error {
	if !all {
		return util.VClusterNameOnlyValidator(cobraCmd, args)
	}

	if len(args) > 0 {
		return fmt.Errorf("%s\nInvalid Args: cannot combine a virtual cluster name (%q) with --all\nRun with --help for more details on arguments", cobraCmd.UseLine(), strings.Join(args, " "))
	}

	return nil
}

// Run executes the functionality
func (cmd *DeleteCmd) Run(cobraCmd *cobra.Command, args []string) error {
	cfg := cmd.LoadedConfig(cmd.log)

	// If driver has been passed as flag use it, otherwise read it from the config file
	driverType, err := config.ParseDriverType(cmp.Or(cmd.Driver, string(cfg.Driver.Type)))
	if err != nil {
		return fmt.Errorf("parse driver type: %w", err)
	}

	ctx := cobraCmd.Context()

	// check if there is a platform client or we skip the info message
	platformClient, err := platform.InitClientFromConfig(ctx, cfg)
	if err == nil {
		config.PrintDriverInfo("delete", driverType, cmd.log)
	}

	if driverType == config.PlatformDriver {
		if cmd.All {
			return cli.DeleteAllPlatform(ctx, platformClient, &cmd.DeleteOptions, cmd.log)
		}

		return cli.DeletePlatform(ctx, platformClient, &cmd.DeleteOptions, args[0], cmd.log)
	}

	// log error if platform flags have been set when using driver helm
	var fs []string
	pfs := flagsdelete.ChangedPlatformFlags(cobraCmd)
	for pf, changed := range pfs {
		if changed {
			fs = append(fs, pf)
		}
	}

	if len(fs) > 0 {
		cmd.log.Fatalf("Following platform flags have been set, which won't have any effect when using driver type %s: %s", driverType, strings.Join(fs, ", "))
	}

	if driverType == config.DockerDriver {
		if cmd.All {
			return cli.DeleteAllDocker(ctx, platformClient, &cmd.DeleteOptions, cmd.GlobalFlags, cmd.log)
		}

		return cli.DeleteDocker(ctx, platformClient, &cmd.DeleteOptions, cmd.GlobalFlags, args[0], cmd.log)
	}

	if cmd.All {
		return cli.DeleteAllHelm(ctx, platformClient, &cmd.DeleteOptions, cmd.GlobalFlags, cmd.log)
	}

	return cli.DeleteHelm(ctx, platformClient, &cmd.DeleteOptions, cmd.GlobalFlags, args[0], cmd.log)
}
