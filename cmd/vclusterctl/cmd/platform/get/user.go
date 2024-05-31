package get

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/api/v4/pkg/product"
	"github.com/loft-sh/log"
	"github.com/loft-sh/log/table"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/platform"
	"github.com/spf13/cobra"
)

// UserCmd holds the lags
type UserCmd struct {
	*flags.GlobalFlags

	log log.Logger

	output string
}

const (
	OutputName = "name"
)

// newUserCmd creates a new command
func newUserCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &UserCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	description := product.ReplaceWithHeader("platform get current-user", `
Returns the currently logged in user

Example:
vcluster platform get current-user
########################################################
	`)
	c := &cobra.Command{
		Use:   "current-user",
		Short: "Retrieves the current logged in user",
		Long:  description,
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, _ []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	c.Flags().StringVarP(&cmd.output, "output", "o", OutputValue, "Output format. One of: (json, yaml, value, name).")

	return c
}

// RunUsers executes the functionality
func (cmd *UserCmd) Run(ctx context.Context) error {
	baseClient, err := platform.InitClientFromConfig(ctx, cmd.LoadedConfig(cmd.log))
	if err != nil {
		return err
	}

	client, err := baseClient.Management()
	if err != nil {
		return err
	}

	userName, teamName, err := platform.GetCurrentUser(ctx, client)
	if err != nil {
		return err
	} else if teamName != nil {
		return errors.New("logged in with a team and not a user")
	}

	switch cmd.output {
	case OutputJSON, OutputYAML:
		currentUser := struct {
			Username    string `json:"username"`
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
			Email       string `json:"email"`
		}{
			Username:    userName.Username,
			Name:        userName.Name,
			DisplayName: userName.DisplayName,
			Email:       userName.Email,
		}

		encodedBytes, err := json.Marshal(currentUser)
		if err != nil {
			return fmt.Errorf("json marshal: %w", err)
		}

		if cmd.output == OutputYAML {
			encodedBytes, err = yaml.JSONToYAML(encodedBytes)
			if err != nil {
				return fmt.Errorf("yaml marshal: %w", err)
			}
		}

		if _, err := os.Stdout.Write(encodedBytes); err != nil {
			return err
		}
	case OutputName:
		if _, err := os.Stdout.WriteString(userName.Username); err != nil {
			return err
		}
	case OutputValue, "":
		header := []string{
			"Username",
			"Kubernetes Name",
			"Display Name",
			"Email",
		}
		values := [][]string{
			{
				userName.Username,
				userName.Name,
				userName.DisplayName,
				userName.Email,
			},
		}

		table.PrintTable(cmd.log, header, values)
	}

	return nil
}
