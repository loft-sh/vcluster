package pro

import (
	"fmt"

	loftctl "github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd"
	loftctlreset "github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd/reset"
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
		LogOutput: globalFlags.LogOutput,
	}

	if globalFlags.Config != "" {
		loftctlGlobalFlags.Config = globalFlags.Config
	} else {
		var err error
		loftctlGlobalFlags.Config, err = pro.GetLoftConfigFilePath()
		if err != nil {
			return nil, fmt.Errorf("failed to get vcluster pro configuration file path: %w", err)
		}
	}

	startCmd, err := NewStartCmd(loftctlGlobalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to create vcluster pro start command: %w", err)
	}

	proCmd.AddCommand(startCmd)
	proCmd.AddCommand(loftctlreset.NewResetCmd(loftctlGlobalFlags))

	return proCmd, nil
}

func NewStartCmd(loftctlGlobalFlags *loftctlflags.GlobalFlags) (*cobra.Command, error) {
	starCmd := loftctl.NewStartCmd(loftctlGlobalFlags)

	configPath, err := pro.GetLoftConfigFilePath()
	if err != nil {
		return nil, fmt.Errorf("failed to get vcluster pro configuration file path: %w", err)
	}

	err = starCmd.Flags().Set("config", configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to set config flag: %w", err)
	}

	err = starCmd.Flags().Set("product", "vcluster-pro")
	if err != nil {
		return nil, fmt.Errorf("failed to set product flag: %w", err)
	}

	err = starCmd.Flags().Set("chart-name", "vcluster-control-plane")
	if err != nil {
		return nil, fmt.Errorf("failed to set chart-name flag: %w", err)
	}

	starCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		version := pro.MinimumVersionTag

		latestVersion, err := pro.LatestCompatibleVersion(cmd.Context())
		if err != nil {
			log.GetInstance().Warnf("failed to get latest compatible version: %v", err)
		} else {
			version = latestVersion
		}

		err = starCmd.Flags().Set("version", version)
		if err != nil {
			return fmt.Errorf("failed to set version flag: %w", err)
		}

		return nil
	}

	return starCmd, nil
}
