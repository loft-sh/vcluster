package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
)

type UICmd struct {
	*flags.GlobalFlags

	Log log.Logger
}

func NewUICmd(globalFlags *flags.GlobalFlags) (*cobra.Command, error) {
	cmd := &UICmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
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
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	return uiCmd, nil
}

func (cmd *UICmd) Run(ctx context.Context) error {
	platformClient, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.Log))
	if err != nil {
		return err
	}

	url := os.Getenv(PlatformURL)
	if url == "" {
		url = platformClient.Config().Platform.Host
	}

	if url == "" {
		return fmt.Errorf("please login first using '%s' or start using '%s'", product.LoginCmd(), product.StartCmd())
	}

	// still open the UI
	err = open.Run(url)
	if errors.Is(err, exec.ErrNotFound) {
		cmd.Log.Warnf("Couldn't open the login page in a browser. No browser found: %v", err)
	} else if err != nil {
		return fmt.Errorf("couldn't open the login page in a browser: %w", err)
	}

	cmd.Log.Infof("If the browser does not open automatically, please navigate to %s", url)

	return nil
}
