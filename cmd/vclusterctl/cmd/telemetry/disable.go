package telemetry

import (
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

type DisableCmd struct {
	*flags.GlobalFlags
	log log.Logger
}

func disable(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &DisableCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "disable",
		Short: "Disables collection of anonymized vcluster telemetry",
		Long: `
#######################################################
############## vcluster telemetry disable #############
#######################################################
Disables collection of anonymized vcluster telemetry.

More information about the collected telmetry is in the
docs: https://www.vcluster.com/docs/advanced-topics/telemetry

#######################################################
	`,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd)
		}}

	return cobraCmd
}

func (cmd *DisableCmd) Run(*cobra.Command) error {
	cfg := config.Read(cmd.Config, cmd.log)
	cfg.TelemetryDisabled = true
	return config.Write(cmd.Config, cfg)
}
