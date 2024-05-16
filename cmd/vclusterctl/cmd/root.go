package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/convert"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/get"
	cmdpro "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform"
	cmdtelemetry "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/telemetry"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/use"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/telemetry"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/loft-sh/vcluster/pkg/util/cliconfig"
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
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			if globalFlags.Silent {
				log.SetLevel(logrus.FatalLevel)
			} else if globalFlags.Debug {
				log.SetLevel(logrus.DebugLevel)
			} else {
				log.SetLevel(logrus.InfoLevel)
			}

			if globalFlags.Config == "" {
				var err error
				globalFlags.Config, err = cliconfig.ConfigFilePath()
				if err != nil {
					log.Fatalf("failed to get vcluster configuration file path: %w", err)
				}
			}
		},
		Long: `vcluster root command`,
	}
}

var globalFlags *flags.GlobalFlags

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := os.Setenv("PRODUCT", "vcluster-pro")
	if err != nil {
		panic(err)
	}

	// start telemetry
	telemetry.StartCLI()

	// start command
	log := log.GetInstance()
	rootCmd, err := BuildRoot(log)
	if err != nil {
		recordAndFlush(err)
		log.Fatalf("error building root: %+v\n", err)
	}

	// Execute command
	err = rootCmd.ExecuteContext(context.Background())
	recordAndFlush(err)
	if err != nil {
		if globalFlags.Debug {
			log.Fatalf("%+v", err)
		}

		log.Fatal(err)
	}
}

// BuildRoot creates a new root command from the
func BuildRoot(log log.Logger) (*cobra.Command, error) {
	rootCmd := NewRootCmd(log)
	persistentFlags := rootCmd.PersistentFlags()
	globalFlags = flags.SetGlobalFlags(persistentFlags, log)

	// Set version for --version flag
	rootCmd.Version = upgrade.GetVersion()

	// add top level commands
	rootCmd.AddCommand(NewConnectCmd(globalFlags))
	rootCmd.AddCommand(NewCreateCmd(globalFlags))
	rootCmd.AddCommand(NewListCmd(globalFlags))
	rootCmd.AddCommand(NewDeleteCmd(globalFlags))
	rootCmd.AddCommand(NewPauseCmd(globalFlags))
	rootCmd.AddCommand(NewResumeCmd(globalFlags))
	rootCmd.AddCommand(NewDisconnectCmd(globalFlags))
	rootCmd.AddCommand(NewUpgradeCmd())
	rootCmd.AddCommand(get.NewGetCmd(globalFlags))
	rootCmd.AddCommand(use.NewUseCmd(globalFlags))
	rootCmd.AddCommand(convert.NewConvertCmd(globalFlags))
	rootCmd.AddCommand(cmdtelemetry.NewTelemetryCmd())
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(NewInfoCmd())

	// add pro commands
	proCmd, err := cmdpro.NewProCmd(globalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to create pro command: %w", err)
	}
	rootCmd.AddCommand(proCmd)
	platformCmd, err := cmdpro.NewPlatformCmd(globalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to create platform command: %w", err)
	}
	rootCmd.AddCommand(platformCmd)

	loginCmd, err := NewLoginCmd(globalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to create login command: %w", err)
	}
	rootCmd.AddCommand(loginCmd)

	logoutCmd, err := NewLogoutCmd(globalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to create logout command: %w", err)
	}
	rootCmd.AddCommand(logoutCmd)

	uiCmd, err := NewUICmd(globalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to create ui command: %w", err)
	}
	rootCmd.AddCommand(uiCmd)

	importCmd, err := NewActivateCmd(globalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to create activate command: %w", err)
	}
	rootCmd.AddCommand(importCmd)

	// add completion command
	err = rootCmd.RegisterFlagCompletionFunc("namespace", newNamespaceCompletionFunc(rootCmd.Context()))
	if err != nil {
		return rootCmd, fmt.Errorf("failed to register completion for namespace: %w", err)
	}

	return rootCmd, nil
}

func recordAndFlush(err error) {
	telemetry.CollectorCLI.RecordCLI(platform.Self, err)
	telemetry.CollectorCLI.Flush()
}
