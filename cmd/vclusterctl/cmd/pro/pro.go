package pro

import (
	"fmt"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/pro/connect"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/procli"
	"github.com/spf13/cobra"
)

func NewProCmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	proCmd := &cobra.Command{
		Use:   "pro",
		Short: "vCluster.Pro subcommands",
		Long: `#######################################################
#################### vcluster pro #####################
#######################################################
		`,
		Args: cobra.NoArgs,
	}

	loftctlGlobalFlags, err := procli.GlobalFlags(globalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pro flags: %w", err)
	}

	startCmd, err := NewStartCmd(loftctlGlobalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to create vcluster pro start command: %w", err)
	}

	proCmd.AddCommand(startCmd)
	proCmd.AddCommand(NewResetCmd(loftctlGlobalFlags))
	proCmd.AddCommand(NewGenerateCmd())
	proCmd.AddCommand(connect.NewConnectCmd(loftctlGlobalFlags))
	proCmd.AddCommand(NewTokenCmd(loftctlGlobalFlags))

	return proCmd, nil
}
