package telemetry

import (
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

type EnableCmd struct {
	*flags.GlobalFlags
	log log.Logger
}

func enable(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &EnableCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "enable",
		Short: "Enables collection of anonymized vcluster telemetry",
		Long: `#######################################################
############### vcluster telemetry enable #############
#######################################################
Enables collection of anonymized vcluster telemetry

More information about the collected telmetry is in the
docs: https://www.vcluster.com/docs/advanced-topics/telemetry

#######################################################
	`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return cmd.Run()
		}}

	return cobraCmd
}

func (cmd *EnableCmd) Run() error {
	cfg := cmd.LoadedConfig(cmd.log)
	cfg.TelemetryDisabled = false
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save vCluster config: %w", err)
	}

	return nil
}
