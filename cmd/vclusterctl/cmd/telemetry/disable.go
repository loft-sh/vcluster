package telemetry

import (
	"github.com/loft-sh/vcluster/cmd/vclusterctl/log"
	"github.com/loft-sh/vcluster/pkg/util/cliconfig"
	"github.com/spf13/cobra"
)

type DisableCmd struct {
	log log.Logger
}

func disable() *cobra.Command {
	cmd := &DisableCmd{
		log: log.GetInstance(),
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
docs: https://www.vcluster.com/docs/telemetry

#######################################################
	`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd)
		}}

	return cobraCmd
}

func (cmd *DisableCmd) Run(cobraCmd *cobra.Command) error {
	c, err := cliconfig.GetConfig()
	if err != nil {
		return err
	}

	c.TelemetryDisabled = true

	err = cliconfig.WriteConfig(c)
	if err != nil {
		return err
	}
	return nil
}
