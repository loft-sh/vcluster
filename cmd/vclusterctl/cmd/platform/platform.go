package platform

import (
	"os"
	"path/filepath"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/add"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/backup"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/connect"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/create"
	cmddelete "github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/delete"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/get"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/list"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/set"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/share"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/sleep"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd/platform/wakeup"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
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
		Args:    cobra.NoArgs,
		Aliases: []string{"pro"},
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			if len(os.Args) > 1 && os.Args[1] == "pro" {
				log.GetInstance().Warnf("The \"vcluster pro\" command is deprecated, please use \"vcluster platform\" instead")
			}

			if globalFlags.Silent {
				log.GetInstance().SetLevel(logrus.FatalLevel)
			} else if globalFlags.Debug {
				log.GetInstance().SetLevel(logrus.DebugLevel)
			} else {
				log.GetInstance().SetLevel(logrus.InfoLevel)
			}
		},
	}
	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}
	defaults, err := defaults.NewFromPath(filepath.Join(home, defaults.ConfigFolder), defaults.ConfigFile)
	if err != nil {
		return nil, err
	}

	startCmd := NewStartCmd(globalFlags)
	destroyCmd := NewDestroyCmd(globalFlags)
	loginCmd := NewCobraLoginCmd(globalFlags)
	logoutCmd := NewLogoutCobraCmd(globalFlags)

	platformCmd.AddCommand(startCmd)
	platformCmd.AddCommand(destroyCmd)
	platformCmd.AddCommand(NewResetCmd(globalFlags))
	platformCmd.AddCommand(add.NewAddCmd(globalFlags))
	platformCmd.AddCommand(NewAccessKeyCmd(globalFlags))
	platformCmd.AddCommand(get.NewGetCmd(globalFlags, defaults))
	platformCmd.AddCommand(connect.NewConnectCmd(globalFlags, defaults))
	platformCmd.AddCommand(list.NewListCmd(globalFlags, defaults))
	platformCmd.AddCommand(set.NewSetCmd(globalFlags, defaults))
	platformCmd.AddCommand(backup.NewBackupCmd(globalFlags))
	platformCmd.AddCommand(wakeup.NewWakeupCmd(globalFlags, defaults))
	platformCmd.AddCommand(sleep.NewSleepCmd(globalFlags, defaults))
	platformCmd.AddCommand(share.NewShareCmd(globalFlags, defaults))
	platformCmd.AddCommand(create.NewCreateCmd(globalFlags, defaults))
	platformCmd.AddCommand(cmddelete.NewDeleteCmd(globalFlags, defaults))
	platformCmd.AddCommand(loginCmd)
	platformCmd.AddCommand(logoutCmd)

	return platformCmd, nil
}
