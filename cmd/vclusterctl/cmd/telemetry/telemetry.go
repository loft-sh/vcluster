package telemetry

import (
	"github.com/spf13/cobra"
)

func NewTelemetryCmd() *cobra.Command {
	telemetryCmd := &cobra.Command{
		Use:   "telemetry",
		Short: "Sets your vcluster telemetry preferences",
		Long: `
#######################################################
################## vcluster telemetry #################
#######################################################
Sets your vcluster telemetry preferences.
Default: enabled.

More information about the collected telmetry is in the
docs: https://www.vcluster.com/docs/advanced-topics/telemetry
	`,
		Args: cobra.NoArgs,
	}

	//TODO: hide global flags on this command and all sub-commands, same for the top-level upgrade command

	telemetryCmd.AddCommand(disable())
	telemetryCmd.AddCommand(enable())
	return telemetryCmd
}
