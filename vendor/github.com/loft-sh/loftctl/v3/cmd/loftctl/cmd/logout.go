package cmd

import (
	"context"
	"fmt"

	"github.com/loft-sh/api/v3/pkg/product"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/upgrade"
	"github.com/loft-sh/log"
	"github.com/mgutz/ansi"
	"github.com/spf13/cobra"
)

// LogoutCmd holds the logout cmd flags
type LogoutCmd struct {
	*flags.GlobalFlags

	Log log.Logger
}

// NewLoginCmd creates a new open command
func NewLogoutCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &LogoutCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	description := product.ReplaceWithHeader("logout", `
Log out of loft

Example:
loft logout
########################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
########################################################
#################### devspace logout ####################
########################################################
Log out of loft

Example:
devspace logout
########################################################
	`
	}

	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: product.Replace("Log out of a loft instance"),
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.RunLogout(cobraCmd.Context(), args)
		},
	}

	return logoutCmd
}

// RunLogin executes the functionality "loft login"
func (cmd *LogoutCmd) RunLogout(ctx context.Context, args []string) error {
	loader, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	config := loader.Config()

	// delete old access key if were logged in before
	if config.AccessKey != "" {
		err := loader.Logout(ctx)
		if err != nil {
			return fmt.Errorf("logout: %w", err)
		}

		configHost := config.Host

		config.Host = ""
		config.AccessKey = ""
		config.LastInstallContext = ""
		config.Insecure = false

		err = loader.Save()
		if err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		cmd.Log.Donef(product.Replace("Successfully logged out of loft instance %s"), ansi.Color(configHost, "white+b"))
	}

	return nil
}
