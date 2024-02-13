package cmd

import (
	"github.com/loft-sh/vcluster/pkg/util/cp"
	"github.com/spf13/cobra"
)

func NewCpCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cp",
		Short: "copy a file",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) (err error) {
			return cp.Cp(args[0], args[1])
		},
	}

	return cmd
}
