package pro

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/spf13/cobra"
)

func NewStartCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Starts the vcluster.pro server",
		Long: `
#######################################################
#################### vcluster pro start #####################
#######################################################
Starts the vcluster pro server
#######################################################
	`,
		DisableFlagParsing: true,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			ctx := cobraCmd.Context()

			cobraCmd.SilenceUsage = true

			log.GetInstance().Info("Starting vcluster pro server ...")

			filePath, version, err := pro.LatestLoftBinary(ctx)
			if err != nil {
				return fmt.Errorf("failed to get latest loft binary: %w", err)
			}

			configFilePath, err := pro.LoftConfigFilePath(version)
			if err != nil {
				return fmt.Errorf("failed to get loft config file path: %w", err)
			}

			workingDir, err := pro.LoftWorkingDirectory(version)
			if err != nil {
				return fmt.Errorf("failed to get loft working directory: %w", err)
			}

			args = append([]string{"start"}, args...)

			cmd := exec.CommandContext(ctx, filePath, args...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			cmd.Dir = workingDir

			cmd.Env = append(cmd.Env, os.Environ()...)
			cmd.Env = append(cmd.Env, fmt.Sprintf("LOFT_CONFIG=%s", configFilePath))
			cmd.Env = append(cmd.Env, "LOFT_VCLUSTER_PRO=true")

			err = cmd.Run()
			if err != nil {
				return fmt.Errorf("failed to start vcluster pro server: %w", err)
			}

			return nil
		},
	}

	return startCmd
}
