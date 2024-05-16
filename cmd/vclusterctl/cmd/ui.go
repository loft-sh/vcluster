package cmd

import (
	"errors"
	"fmt"

	loftctl "github.com/loft-sh/loftctl/v4/cmd/loftctl/cmd"
	loftctlflags "github.com/loft-sh/loftctl/v4/cmd/loftctl/flags"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/spf13/cobra"
)

func NewUICmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	cmd := &loftctl.UiCmd{
		GlobalFlags: &loftctlflags.GlobalFlags{
			Config:    globalFlags.Config,
			LogOutput: globalFlags.LogOutput,
			Silent:    globalFlags.Silent,
			Debug:     globalFlags.Debug,
		},
		Log: log.GetInstance(),
	}

	description := `########################################################
##################### vcluster ui ######################
########################################################
Open the vCluster platform web UI

Example:
vcluster ui
########################################################
	`

	uiCmd := &cobra.Command{
		Use:   "ui",
		Short: "Start the web UI",
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			err := cmd.Run(cobraCmd.Context(), args)
			if err != nil {
				if errors.Is(err, loftctl.ErrNoUrl) {
					return fmt.Errorf("%w: please login first using 'vcluster login' or start using 'vcluster pro start'", err)
				}

				return fmt.Errorf("failed to run ui command: %w", err)
			}

			return nil
		},
	}

	return uiCmd, nil
}
