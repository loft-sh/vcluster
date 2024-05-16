package platform

import (
	"fmt"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/add"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

func NewPlatformCmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	platformCmd := &cobra.Command{
		Use:   "platform",
		Short: "vCluster platform subcommands",
		Long: `#######################################################
################## vcluster platform ##################
#######################################################
		`,
		Args: cobra.NoArgs,
	}

	startCmd, err := NewStartCmd(globalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to create vcluster platform start command: %w", err)
	}

	platformCmd.AddCommand(startCmd)
	platformCmd.AddCommand(NewResetCmd(globalFlags))
	platformCmd.AddCommand(add.NewAddCmd(globalFlags))
	platformCmd.AddCommand(NewAccessKeyCmd(globalFlags))
	platformCmd.AddCommand(NewImportCmd(globalFlags))

	return platformCmd, nil
}
