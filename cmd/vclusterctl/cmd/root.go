package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/certs"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/credits"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/debug"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/node"
	cmdplatform "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/set"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/registry"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/snapshot"
	cmdtelemetry "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/telemetry"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/token"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/use"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const compCmdName = "completion"

// NewRootCmd returns a new root command
func NewRootCmd(log log.Logger) *cobra.Command {
	return &cobra.Command{
		Use:           "vcluster",
		SilenceUsage:  true,
		SilenceErrors: true,
		Short:         "Welcome to vcluster!",
		PersistentPreRunE: func(cobraCmd *cobra.Command, _ []string) error {
			if globalFlags == nil {
				return errors.New("nil globalFlags")
			}

			if globalFlags.Config == "" {
				var err error
				globalFlags.Config, err = config.DefaultFilePath()
				if err != nil {
					log.Fatalf("failed to get vcluster configuration file path: %w", err)
				}
			}

			// start telemetry — skip for completion commands
			if !isCompletionCommand(cobraCmd) {
				telemetry.StartCLI(globalFlags.LoadedConfig(log))
			}

			if globalFlags.Silent {
				log.SetLevel(logrus.FatalLevel)
			} else if globalFlags.Debug {
				log.SetLevel(logrus.DebugLevel)
			} else {
				log.SetLevel(logrus.InfoLevel)
			}

			return nil
		},
		Long: `vcluster root command`,
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := os.Setenv("PRODUCT", "vcluster-pro")
	if err != nil {
		panic(err)
	}

	// start command
	log := log.GetInstance()
	rootCmd, globalFlags, err := BuildRoot(log)
	if err != nil {
		log.Fatalf("error building root: %+v\n", err)
	}

	// Execute command
	ctx := context.Background()
	if requested, helpCmd := helpRequested(rootCmd, os.Args[1:]); requested {
		err = executeHelp(ctx, rootCmd, helpCmd)
	} else {
		err = rootCmd.ExecuteContext(ctx)
	}
	recordAndFlush(err, log, globalFlags)
	if err != nil {
		if globalFlags != nil && globalFlags.Debug {
			log.Fatalf("%+v", err)
		}

		log.Fatal(err)
	}
}

// helpRequested reports whether args contain a bare "-h"/"--help" that a value
// taking flag would otherwise swallow, and the command to print help for. args
// are the arguments without the binary name, exactly what cobra's Find expects.
func helpRequested(rootCmd *cobra.Command, args []string) (bool, *cobra.Command) {
	if len(args) == 0 || args[0] == cobra.ShellCompRequestCmd || args[0] == cobra.ShellCompNoDescRequestCmd {
		return false, nil
	}

	// everything after the "--" terminator is positional
	if i := slices.Index(args, "--"); i >= 0 {
		args = args[:i]
	}
	if !slices.Contains(args, "-h") && !slices.Contains(args, "--help") {
		return false, nil
	}

	cmd, _, err := rootCmd.Find(args)
	if err != nil {
		// unknown command, let cobra report it
		return false, nil
	}

	return true, cmd
}

// executeHelp prints helpCmd's help by handing its path plus "--help" back to
// cobra, so the output goes through the same initialization as a help flag
// cobra parsed itself. Cobra stops before PersistentPreRunE, so nothing runs.
func executeHelp(ctx context.Context, rootCmd, helpCmd *cobra.Command) error {
	rootCmd.SetArgs(append(commandArgs(helpCmd), "--help"))

	return rootCmd.ExecuteContext(ctx)
}

// commandArgs returns the args that reach cmd, without the binary name. Command
// names cannot contain spaces, so splitting the path is unambiguous.
func commandArgs(cmd *cobra.Command) []string {
	return strings.Fields(cmd.CommandPath())[1:]
}

var globalFlags *flags.GlobalFlags

// BuildRoot creates a new root command from the
func BuildRoot(log log.Logger) (*cobra.Command, *flags.GlobalFlags, error) {
	rootCmd := NewRootCmd(log)
	persistentFlags := rootCmd.PersistentFlags()
	globalFlags = flags.SetGlobalFlags(persistentFlags, log)

	home, err := homedir.Dir()
	if err != nil {
		return nil, nil, err
	}
	defaults, err := defaults.NewFromPath(filepath.Join(home, defaults.ConfigFolder), defaults.ConfigFile)
	if err != nil {
		log.Debugf("Error loading defaults: %v", err)
		return nil, nil, err
	}

	// Set version for --version flag
	rootCmd.Version = upgrade.GetVersion()

	// add top level commands
	rootCmd.AddCommand(NewConnectCmd(globalFlags))
	rootCmd.AddCommand(NewCreateCmd(globalFlags))
	rootCmd.AddCommand(NewListCmd(globalFlags))
	rootCmd.AddCommand(NewDescribeCmd(globalFlags, defaults))
	rootCmd.AddCommand(NewDeleteCmd(globalFlags))
	rootCmd.AddCommand(NewPauseCmd(globalFlags))
	rootCmd.AddCommand(NewResumeCmd(globalFlags))
	rootCmd.AddCommand(NewDisconnectCmd(globalFlags))
	rootCmd.AddCommand(NewUpgradeCmd())
	rootCmd.AddCommand(snapshot.NewSnapshot(globalFlags))
	rootCmd.AddCommand(NewRestore(globalFlags))
	rootCmd.AddCommand(use.NewUseCmd(globalFlags))
	rootCmd.AddCommand(debug.NewDebugCommand(globalFlags))
	rootCmd.AddCommand(cmdtelemetry.NewTelemetryCmd(globalFlags))
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(NewInfoCmd(globalFlags))
	rootCmd.AddCommand(set.NewSetCmd(globalFlags, defaults))
	rootCmd.AddCommand(token.NewTokenCmd(globalFlags))
	rootCmd.AddCommand(node.NewNodeCmd(globalFlags))
	rootCmd.AddCommand(registry.NewRegistryCmd(globalFlags))
	rootCmd.AddCommand(certs.NewCertsCmd(globalFlags))

	// add platform commands
	platformCmd, err := cmdplatform.NewPlatformCmd(globalFlags)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create platform command: %w", err)
	}
	rootCmd.AddCommand(platformCmd)

	loginCmd, err := NewLoginCmd(globalFlags)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create login command: %w", err)
	}
	rootCmd.AddCommand(loginCmd)

	logoutCmd, err := NewLogoutCmd(globalFlags)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create logout command: %w", err)
	}
	rootCmd.AddCommand(logoutCmd)

	uiCmd, err := NewUICmd(globalFlags)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create ui command: %w", err)
	}
	rootCmd.AddCommand(uiCmd)
	rootCmd.AddCommand(credits.NewCreditsCmd())

	// add completion command
	err = rootCmd.RegisterFlagCompletionFunc("namespace", completion.NewNamespaceCompletionFunc(rootCmd.Context()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to register completion for namespace: %w", err)
	}

	return rootCmd, globalFlags, nil
}

func recordAndFlush(err error, log log.Logger, globalFlags *flags.GlobalFlags) {
	if globalFlags == nil {
		panic("empty global flags")
	}

	telemetry.CollectorCLI.RecordCLI(globalFlags.LoadedConfig(log), platform.Self, err)
	telemetry.CollectorCLI.Flush()
}

func isCompletionCommand(cmd *cobra.Command) bool {
	name := cmd.Name()
	if name == cobra.ShellCompRequestCmd || name == cobra.ShellCompNoDescRequestCmd {
		return true
	}
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == compCmdName {
			return true
		}
	}
	return false
}
