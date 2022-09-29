package cmd

import (
	"github.com/spf13/cobra"
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

// BuildRoot creates a new root command from the
func BuildRoot() *cobra.Command {
	rootCmd := NewRootCmd()

	// add top level commands
	rootCmd.AddCommand(NewStartCommand())
	rootCmd.AddCommand(NewCertsCommand())
	rootCmd.AddCommand(NewHostpathMapperCommand())
	return rootCmd
}
