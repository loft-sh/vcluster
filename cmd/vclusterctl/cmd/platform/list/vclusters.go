package list

import (
	"context"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

// VClustersCmd holds the login cmd flags
type VClustersCmd struct {
	*flags.GlobalFlags
	cli.ListOptions

	log log.Logger
}

// newVClustersCmd creates a new command
func newVClustersCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &VClustersCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}

	cobraCmd := &cobra.Command{
		Use:   "vclusters",
		Short: "Lists all virtual clusters that are connected to the current platform",
		Long: `##########################################################################
#################### vcluster platform list vclusters ####################
##########################################################################
Lists all virtual clusters that are connected to the current platform

Example:
vcluster platform list vclusters
##########################################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Output, "output", "table", "Choose the format of the output. [table|json]")

	return cobraCmd
}

func (cmd *VClustersCmd) Run(ctx context.Context) error {
	return cli.ListPlatform(ctx, &cmd.ListOptions, cmd.GlobalFlags, cmd.log)
}
