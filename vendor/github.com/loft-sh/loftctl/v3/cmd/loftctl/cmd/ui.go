package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/log"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
)

var (
	ErrNoUrl = errors.New("no url found")
)

// UiCmd holds the ui cmd flags
type UiCmd struct {
	*flags.GlobalFlags

	Log log.Logger
}

func NewUiCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &UiCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("ui", `
Open the loft management UI in the browser

Example:
loft ui
########################################################
	`)

	uiCmd := &cobra.Command{
		Use:   "ui",
		Short: product.Replace("Open the loft management UI in the browser"),
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	return uiCmd
}

func (cmd *UiCmd) Run(ctx context.Context, args []string) error {
	loader, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	url := os.Getenv(LoftUrl)
	if url == "" {
		url = loader.Config().Host
	}

	if url == "" {
		return fmt.Errorf("%w: please login first using '%s' or start using '%s'", ErrNoUrl, product.LoginCmd(), product.StartCmd())
	}

	err = open.Run(url)
	if err != nil {
		return fmt.Errorf("error opening url: %w", err)
	}

	return nil
}
