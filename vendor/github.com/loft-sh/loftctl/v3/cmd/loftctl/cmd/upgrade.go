package cmd

import (
	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// UpgradeCmd is a struct that defines a command call for "upgrade"
type UpgradeCmd struct {
	log     log.Logger
	Version string
}

// NewUpgradeCmd creates a new upgrade command
func NewUpgradeCmd() *cobra.Command {
	cmd := &UpgradeCmd{
		log: log.GetInstance(),
	}

	upgradeCmd := &cobra.Command{
		Use:   "upgrade",
		Short: product.Replace("Upgrade the loft CLI to the newest version"),
		Long: product.ReplaceWithHeader("upgrade", `
Upgrades the loft CLI to the newest version
########################################################`),
		Args: cobra.NoArgs,
		RunE: cmd.Run,
	}

	upgradeCmd.Flags().StringVar(&cmd.Version, "version", "", product.Replace("The version to update loft to. Defaults to the latest stable version available"))
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
