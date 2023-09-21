package vclusterpro

import (
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/generate"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/reset"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/vcluster-pro/secret"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/vcluster-pro/space"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/defaults"
	"github.com/spf13/cobra"
)

func BuildVclusterProRoot(rootCmd *cobra.Command, globalFlags *flags.GlobalFlags, defaults *defaults.Defaults, additionalCommands []*cobra.Command) {
	// vcluster pro related top level commands
	rootCmd.AddCommand(NewConnectCmd(globalFlags, defaults))
	rootCmd.AddCommand(NewCreateCmd(globalFlags, defaults))
	rootCmd.AddCommand(NewDeleteCmd(globalFlags, defaults))
	rootCmd.AddCommand(NewImportCmd(globalFlags, defaults))
	rootCmd.AddCommand(NewListCmd(globalFlags, defaults))

	// add subcommands
	rootCmd.AddCommand(space.NewRootCmd(globalFlags, defaults))
	rootCmd.AddCommand(secret.NewRootCmd(globalFlags, defaults))
	rootCmd.AddCommand(generate.NewGenerateCmd(globalFlags))
	rootCmd.AddCommand(reset.NewResetCmd(globalFlags))

	// add additional commands
	rootCmd.AddCommand(additionalCommands...)
}
