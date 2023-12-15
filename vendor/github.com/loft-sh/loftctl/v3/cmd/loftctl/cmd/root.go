package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/connect"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/create"
	cmddefaults "github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/defaults"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/delete"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/devpod"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/generate"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/get"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/importcmd"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/list"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/reset"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/set"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/share"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/sleep"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/use"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/vars"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/wakeup"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewRootCmd returns a new root command
func NewRootCmd(streamLogger *log.StreamLogger) *cobra.Command {
	return &cobra.Command{
		Use:           "loft",
		SilenceUsage:  true,
		SilenceErrors: true,
		Short:         product.Replace("Welcome to Loft!"),
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if globalFlags.Silent {
				streamLogger.SetLevel(logrus.FatalLevel)
			}
			if globalFlags.Config == "" && os.Getenv("LOFT_CONFIG") != "" {
				globalFlags.Config = os.Getenv("LOFT_CONFIG")
			}

			if globalFlags.LogOutput == "json" {
				streamLogger.SetFormat(log.JSONFormat)
			} else if globalFlags.LogOutput == "raw" {
				streamLogger.SetFormat(log.RawFormat)
			} else if globalFlags.LogOutput != "plain" {
				return fmt.Errorf("unrecognized log format %s, needs to be either plain or json", globalFlags.LogOutput)
			}

			return nil
		},
		Long: product.Replace(`Loft CLI`) + " - www.loft.sh",
	}
}

var globalFlags *flags.GlobalFlags

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	log := log.Default
	rootCmd := BuildRoot(log)

	// Set version for --version flag
	rootCmd.Version = upgrade.GetVersion()

	// Execute command
	err := rootCmd.ExecuteContext(context.Background())
	if err != nil {
		if globalFlags.Debug {
			log.Fatalf("%+v", err)
		} else {
			log.Fatal(err)
		}
	}
}

// BuildRoot creates a new root command from the
func BuildRoot(log *log.StreamLogger) *cobra.Command {
	rootCmd := NewRootCmd(log)
	persistentFlags := rootCmd.PersistentFlags()
	globalFlags = flags.SetGlobalFlags(persistentFlags)
	defaults, err := defaults.NewFromPath(defaults.ConfigFolder, defaults.ConfigFile)
	if err != nil {
		log.Debugf("Error loading defaults: %v", err)
	}

	// add top level commands
	rootCmd.AddCommand(NewStartCmd(globalFlags))
	rootCmd.AddCommand(NewLoginCmd(globalFlags))
	rootCmd.AddCommand(NewLogoutCmd(globalFlags))
	rootCmd.AddCommand(NewUiCmd(globalFlags))
	rootCmd.AddCommand(NewTokenCmd(globalFlags))
	rootCmd.AddCommand(NewBackupCmd(globalFlags))
	rootCmd.AddCommand(NewCompletionCmd(rootCmd, globalFlags))
	rootCmd.AddCommand(NewUpgradeCmd())

	// add subcommands
	rootCmd.AddCommand(list.NewListCmd(globalFlags))
	rootCmd.AddCommand(use.NewUseCmd(globalFlags, defaults))
	rootCmd.AddCommand(create.NewCreateCmd(globalFlags, defaults))
	rootCmd.AddCommand(delete.NewDeleteCmd(globalFlags, defaults))
	rootCmd.AddCommand(generate.NewGenerateCmd(globalFlags))
	rootCmd.AddCommand(get.NewGetCmd(globalFlags, defaults))
	rootCmd.AddCommand(vars.NewVarsCmd(globalFlags))
	rootCmd.AddCommand(share.NewShareCmd(globalFlags, defaults))
	rootCmd.AddCommand(set.NewSetCmd(globalFlags, defaults))
	rootCmd.AddCommand(reset.NewResetCmd(globalFlags))
	rootCmd.AddCommand(sleep.NewSleepCmd(globalFlags, defaults))
	rootCmd.AddCommand(wakeup.NewWakeUpCmd(globalFlags, defaults))
	rootCmd.AddCommand(importcmd.NewImportCmd(globalFlags))
	rootCmd.AddCommand(connect.NewConnectCmd(globalFlags))
	rootCmd.AddCommand(cmddefaults.NewDefaultsCmd(globalFlags, defaults))
	rootCmd.AddCommand(devpod.NewDevPodCmd(globalFlags))

	return rootCmd
}
