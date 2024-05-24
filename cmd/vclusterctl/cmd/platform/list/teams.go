package list

import (
	"context"

	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform/clihelper"
	client "github.com/loft-sh/vcluster/pkg/platform/loftclient"
	"github.com/loft-sh/vcluster/pkg/platform/loftclient/helper"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// TeamsCmd holds the cmd flags
type TeamsCmd struct {
	*flags.GlobalFlags

	log log.Logger
}

// NewTeamsCmd creates a new command
func NewTeamsCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &TeamsCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("list teams", `
List the loft teams you are member of

Example:
loft list teams
########################################################
	`)
	clustersCmd := &cobra.Command{
		Use:   "teams",
		Short: product.Replace("Lists the loft teams you are member of"),
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	return clustersCmd
}

// RunUsers executes the functionality "loft list users"
func (cmd *TeamsCmd) Run(ctx context.Context) error {
	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	client, err := baseClient.Management()
	if err != nil {
		return err
	}

	userName, teamName, err := helper.GetCurrentUser(ctx, client)
	if err != nil {
		return err
	} else if teamName != nil {
		return errors.New("logged in as a team")
	}

	header := []string{
		"Name",
	}
	values := [][]string{}
	for _, team := range userName.Teams {
		values = append(values, []string{
			clihelper.DisplayName(team),
		})
	}

	table.PrintTable(cmd.log, header, values)
	return nil
}
