package cmd

import (
	"errors"
	"fmt"

	loftctl "github.com/loft-sh/loftctl/v3/cmd/loftctl/cmd"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/flags"
	"github.com/loft-sh/vcluster/pkg/procli"
	"github.com/spf13/cobra"
)

func NewUICmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	loftctlGlobalFlags, err := procli.GlobalFlags(globalFlags)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pro flags: %w", err)
	}

	cmd := &loftctl.UiCmd{
		GlobalFlags: loftctlGlobalFlags,
		Log:         log.GetInstance(),
	}

	description := `########################################################
##################### vcluster ui ######################
########################################################
Open the vCluster.Pro web UI

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
