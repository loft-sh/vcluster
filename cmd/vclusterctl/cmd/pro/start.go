package pro

import (
	"fmt"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/spf13/cobra"
)

type StartCmd struct{}

func NewStartCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := StartCmd{}

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
		RunE:               cmd.RunE,
	}

	return startCmd
}

func (sc *StartCmd) RunE(cobraCmd *cobra.Command, args []string) error {
	ctx := cobraCmd.Context()

	cobraCmd.SilenceUsage = true

	log.GetInstance().Info("Starting vcluster pro server ...")

	args = append([]string{"start"}, args...)

	err := pro.RunLoftCli(ctx, "latest", args)
	if err != nil {
		return fmt.Errorf("failed to start vcluster pro server: %w", err)
	}

	return nil
}
