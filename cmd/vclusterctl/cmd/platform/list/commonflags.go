package list

import (
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/spf13/cobra"
)

func AddCommonFlags(cmd *cobra.Command, options *cli.ListOptions) {
	cmd.Flags().StringVar(&options.Output, "output", "table", "Choose the format of the output. [table|json]")
}
