package cmd

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	loftlogr "github.com/loft-sh/log/logr"
	"github.com/loft-sh/vcluster/cmd/vcluster/cmd/debug"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewRootCmd returns a new root command
func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "vcluster",
		SilenceUsage:  true,
		SilenceErrors: true,
		Short:         "Welcome to vcluster!",
		Long:          `vcluster root command`,
	}
}

func RunRoot() {
	// set global logger
	if os.Getenv("DEBUG") == "true" {
		_ = os.Setenv("LOFT_LOG_LEVEL", "debug")
	} else {
		_ = os.Setenv("LOFT_LOG_LEVEL", "info")
	}

	// set global logger
	logger, err := loftlogr.NewLoggerWithOptions(
		loftlogr.WithOptionsFromEnv(),
		loftlogr.WithComponentName("vcluster"),
		loftlogr.WithGlobalZap(true),
		loftlogr.WithGlobalKlog(true),
	)
	if err != nil {
		klog.Fatal(err)
	}
	ctrl.SetLogger(logger)
	ctx := logr.NewContext(context.Background(), logger)

	// create a new command and execute
	err = BuildRoot().ExecuteContext(ctx)
	if err != nil {
		klog.FromContext(ctx).Error(err, "error")
		os.Exit(1)
	}
}

// BuildRoot creates a new root command from the
func BuildRoot() *cobra.Command {
	rootCmd := NewRootCmd()

	// add top level commands
	rootCmd.AddCommand(NewStartCommand())
	rootCmd.AddCommand(NewCpCommand())
	rootCmd.AddCommand(NewPortForwardCommand())
	rootCmd.AddCommand(debug.NewDebugCmd())
	return rootCmd
}
