//go:build pro
// +build pro

package pro

import (
	"fmt"

	loftctl "github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd"
	loftctlflags "github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/pro"
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

	loftctlGlobalFlags := &loftctlflags.GlobalFlags{
		Silent:    globalFlags.Silent,
		Debug:     globalFlags.Debug,
		Config:    globalFlags.Config,
		LogOutput: globalFlags.LogOutput,
	}

	startCmd, err := NewStartCmd(loftctlGlobalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to create vcluster pro start command: %w", err)
	}

	proCmd.AddCommand(startCmd)

	return proCmd, nil
}

func NewStartCmd(loftctlGlobalFlags *loftctlflags.GlobalFlags) (*cobra.Command, error) {
	starCmd := loftctl.NewStartCmd(loftctlGlobalFlags)

	configPath, err := pro.GetLoftConfigFilePath()
	if err != nil {
		return nil, fmt.Errorf("failed to get vcluster pro configuration file path: %w", err)
	}

	starCmd.Flags().Set("config", configPath)

	starCmd.Flags().Set("product", "vcluster-pro")
	starCmd.Flags().Set("chart-name", "vcluster-control-plane")

	starCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		version := pro.MinimumVersionTag

		latestVersion, err := pro.LatestCompatibleVersion(cmd.Context())
		if err != nil {
			log.GetInstance().Warnf("failed to get latest compatible version: %v", err)
		} else {
			version = latestVersion
		}

		starCmd.Flags().Set("version", version)
	}

	return starCmd, nil
}
