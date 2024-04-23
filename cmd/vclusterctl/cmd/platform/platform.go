package platform

import (
	"fmt"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/connect"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/spf13/cobra"
)

func NewPlatformCmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	platformCmd := &cobra.Command{
		Use:   "platform",
		Short: "vCluster platform subcommands",
		Long: `#######################################################
################## vcluster platform ##################
#######################################################

Deprecated, please use vcluster platform instead
		`,
		Args: cobra.NoArgs,
	}

	loftctlGlobalFlags, err := platform.GlobalFlags(globalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pro flags: %w", err)
	}

	startCmd, err := NewStartCmd(loftctlGlobalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to create vcluster platform start command: %w", err)
	}

	platformCmd.AddCommand(startCmd)
	platformCmd.AddCommand(NewResetCmd(loftctlGlobalFlags))
	platformCmd.AddCommand(connect.NewConnectCmd(loftctlGlobalFlags))
	platformCmd.AddCommand(NewAccessKeyCmd(loftctlGlobalFlags))
	platformCmd.AddCommand(NewImportCmd(globalFlags))

	return platformCmd, nil
}
