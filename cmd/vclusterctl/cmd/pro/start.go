package pro

import (
	"fmt"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

type StartCmd struct{}

func NewStartCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := StartCmd{}

	startCmd := &cobra.Command{
		Use:                "start",
		Short:              "Starts the vcluster.pro server",
		DisableFlagParsing: true,
		RunE:               cmd.RunE,
	}

	return startCmd
}

func (*StartCmd) RunE(cobraCmd *cobra.Command, args []string) error {
	ctx := cobraCmd.Context()

	cobraCmd.SilenceUsage = true

	containsHelmFlags := lo.ContainsBy(args, func(item string) bool {
		return item == "--chart-name" || item == "--chart-path"
	})

	if !containsHelmFlags {
		args = append(args, "--chart-name", "vcluster-control-plane")
	}

	args = append([]string{"start"}, args...)

	err := pro.RunLoftCli(ctx, "latest", args)
	if err != nil {
		return fmt.Errorf("failed to start vcluster pro server: %w", err)
	}

	return nil
}
