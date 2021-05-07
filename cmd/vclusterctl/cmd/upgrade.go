package cmd

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/upgrade"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// UpgradeCmd is a struct that defines a command call for "upgrade"
type UpgradeCmd struct {
	Version string

	log log.Logger
}

// NewUpgradeCmd creates a new upgrade command
func NewUpgradeCmd() *cobra.Command {
	cmd := &UpgradeCmd{
		log: log.GetInstance(),
	}

	upgradeCmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade the vcluster CLI to the newest version",
		Long: `
#######################################################
################## vcluster upgrade ###################
#######################################################
Upgrades the vcluster CLI to the newest version
#######################################################`,
		Args: cobra.NoArgs,
		RunE: cmd.Run,
	}

	upgradeCmd.Flags().StringVar(&cmd.Version, "version", "", "The version to update vcluster to. Defaults to the latest stable version available")
	return upgradeCmd
}

// Run executes the command logic
func (cmd *UpgradeCmd) Run(cobraCmd *cobra.Command, args []string) error {
	err := upgrade.Upgrade(cmd.Version, cmd.log)
	if err != nil {
		return errors.Errorf("Couldn't upgrade: %v", err)
	}

	return nil
}
