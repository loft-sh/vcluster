package platform

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/add"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/backup"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/connect"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/get"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/list"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/set"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/spf13/cobra"
)

func NewPlatformCmd(globalFlags *flags.GlobalFlags, cfg *config.CLI) (*cobra.Command, error) {
	platformCmd := &cobra.Command{
		Use:   "platform",
		Short: "vCluster platform subcommands",
		Long: `#######################################################
################## vcluster platform ##################
#######################################################
		`,
		Args: cobra.NoArgs,
	}
	defaults, err := defaults.NewFromPath(platform.CacheFolder, defaults.ConfigFile)
	if err != nil {
		return nil, err
	}

	startCmd := NewStartCmd(globalFlags)

	platformCmd.AddCommand(startCmd)
	platformCmd.AddCommand(NewResetCmd(globalFlags))
	platformCmd.AddCommand(add.NewAddCmd(globalFlags))
	platformCmd.AddCommand(NewAccessKeyCmd(globalFlags))
	platformCmd.AddCommand(NewImportCmd(globalFlags))
	platformCmd.AddCommand(get.NewGetCmd(globalFlags, defaults, cfg))
	platformCmd.AddCommand(connect.NewConnectCmd(globalFlags, cfg))
	platformCmd.AddCommand(list.NewListCmd(globalFlags, cfg))
	platformCmd.AddCommand(set.NewSetCmd(globalFlags, defaults, cfg))
	platformCmd.AddCommand(backup.NewBackupCmd(globalFlags, cfg))

	return platformCmd, nil
}
