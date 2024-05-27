package platform

import (
	"fmt"

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

	startCmd, err := NewStartCmd(globalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to create vcluster pro start command: %w", err)
	}

	proCmd.AddCommand(startCmd)
	proCmd.AddCommand(NewResetCmd(globalFlags))
	proCmd.AddCommand(connect.NewConnectCmd(globalFlags))
	proCmd.AddCommand(NewAccessKeyCmd(globalFlags))

	return proCmd, nil
}
