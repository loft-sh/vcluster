package list

import (
	"context"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	pdefaults "github.com/loft-sh/vcluster/pkg/platform/defaults"
	"github.com/spf13/cobra"
)

// VClustersCmd holds the login cmd flags
type VClustersCmd struct {
	*flags.GlobalFlags
	cli.ListOptions

	log     log.Logger
	Project string
}

// newVClustersCmd creates a new command
func newVClustersCmd(globalFlags *flags.GlobalFlags, defaults *pdefaults.Defaults) *cobra.Command {
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

	p, _ := defaults.Get(pdefaults.KeyProject, "")
	cobraCmd.Flags().StringVarP(&cmd.Project, "project", "p", p, "The project to use")
	cobraCmd.Flags().StringVar(&cmd.Output, "output", "table", "Choose the format of the output. [table|json]")

	return cobraCmd
}

func (cmd *VClustersCmd) Run(ctx context.Context) error {
	return cli.ListPlatform(ctx, &cmd.ListOptions, cmd.GlobalFlags, cmd.log, cmd.Project)
}
