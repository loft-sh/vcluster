package platform

import (
	"fmt"

	loftctlflags "github.com/loft-sh/loftctl/v4/cmd/loftctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/connect"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

func NewProCmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	proCmd := &cobra.Command{
		Use:   "pro",
		Short: "vCluster platform subcommands",
		Long: `#######################################################
#################### vcluster pro #####################
#######################################################

Deprecated, please use vcluster platform instead
		`,
		Args: cobra.NoArgs,
	}

	loftctlGlobalFlags := &loftctlflags.GlobalFlags{
		Config:    globalFlags.Config,
		LogOutput: globalFlags.LogOutput,
		Silent:    globalFlags.Silent,
		Debug:     globalFlags.Debug,
	}

	startCmd, err := NewStartCmd(loftctlGlobalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to create vcluster pro start command: %w", err)
	}

	proCmd.AddCommand(startCmd)
	proCmd.AddCommand(NewResetCmd(loftctlGlobalFlags))
	proCmd.AddCommand(connect.NewConnectCmd(loftctlGlobalFlags))
	proCmd.AddCommand(NewAccessKeyCmd(loftctlGlobalFlags))

	return proCmd, nil
}
