package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/debug"
	"github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/mitchellh/go-homedir"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/convert"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/credits"
	cmdplatform "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/set"
	cmdtelemetry "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/telemetry"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/use"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewRootCmd returns a new root command
func NewRootCmd(log log.Logger) *cobra.Command {
	return &cobra.Command{
		Use:           "vcluster",
		SilenceUsage:  true,
		SilenceErrors: true,
		Short:         "Welcome to vcluster!",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
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

			// start telemetry
			telemetry.StartCLI(globalFlags.LoadedConfig(log))

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
	err = rootCmd.ExecuteContext(context.Background())
	recordAndFlush(err, log, globalFlags)
	if err != nil {
		if globalFlags != nil && globalFlags.Debug {
			log.Fatalf("%+v", err)
		}

		log.Fatal(err)
	}
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
	rootCmd.AddCommand(use.NewUseCmd(globalFlags))
	rootCmd.AddCommand(debug.NewDebugCommand(globalFlags))
	rootCmd.AddCommand(convert.NewConvertCmd(globalFlags))
	rootCmd.AddCommand(cmdtelemetry.NewTelemetryCmd(globalFlags))
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(NewInfoCmd(globalFlags))
	rootCmd.AddCommand(set.NewSetCmd(globalFlags, defaults))

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
