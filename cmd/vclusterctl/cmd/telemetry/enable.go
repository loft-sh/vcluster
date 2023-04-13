package telemetry

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/util/cliconfig"
	"github.com/spf13/cobra"
)

type EnableCmd struct {
	log log.Logger
}

func enable() *cobra.Command {
	cmd := &EnableCmd{
		log: log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "enable",
		Short: "Enables collection of anonymized vcluster telemetry",
		Long: `
#######################################################
############### vcluster telemetry enable #############
#######################################################
Enables collection of anonymized vcluster telemetry

More information about the collected telmetry is in the
docs: https://www.vcluster.com/docs/telemetry

#######################################################
	`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd)
		}}

	return cobraCmd
}

func (cmd *EnableCmd) Run(cobraCmd *cobra.Command) error {
	c, err := cliconfig.GetConfig()
	if err != nil {
		return err
	}

	c.TelemetryDisabled = false

	err = cliconfig.WriteConfig(c)
	if err != nil {
		return err
	}
	return nil
}
