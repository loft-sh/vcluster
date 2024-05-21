package get

import (
	"context"
	"fmt"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/loftctl/v4/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v4/pkg/client"
	"github.com/loft-sh/loftctl/v4/pkg/client/helper"
	"github.com/loft-sh/loftctl/v4/pkg/projectutil"
	"github.com/loft-sh/loftctl/v4/pkg/upgrade"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// UserCmd holds the lags
type UserCmd struct {
	*flags.GlobalFlags

	log log.Logger
}

// NewUserCmd creates a new command
func NewUserCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &UserCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("get user", `
Returns the currently logged in user

Example:
loft get user
########################################################
	`)
	if upgrade.IsPlugin == "true" {
		description = `
########################################################
################## devspace get user ###################
########################################################
Returns the currently logged in user

Example:
devspace get user
########################################################
	`
	}
	c := &cobra.Command{
		Use:   "user",
		Short: "Retrieves the current logged in user",
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	return c
}

// RunUsers executes the functionality
func (cmd *UserCmd) Run(ctx context.Context) error {
	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}
	self, err := baseClient.GetSelf(ctx)
	if err != nil {
		return fmt.Errorf("failed to get self: %w", err)
	}
	projectutil.SetProjectNamespacePrefix(self.Status.ProjectNamespacePrefix)

	client, err := baseClient.Management()
	if err != nil {
		return err
	}

	userName, teamName, err := helper.GetCurrentUser(ctx, client)
	if err != nil {
		return err
	} else if teamName != nil {
		return errors.New("logged in with a team and not a user")
	}

	header := []string{
		"Username",
		"Kubernetes Name",
		"Display Name",
		"Email",
	}
	values := [][]string{}
	values = append(values, []string{
		userName.Username,
		userName.Name,
		userName.DisplayName,
		userName.Email,
	})

	table.PrintTable(cmd.log, header, values)
	return nil
}
