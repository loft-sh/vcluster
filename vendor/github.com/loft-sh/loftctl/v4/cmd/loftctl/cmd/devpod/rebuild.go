package devpod

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/loft-sh/loftctl/v4/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v4/pkg/client"
	devpodpkg "github.com/loft-sh/loftctl/v4/pkg/devpod"
	"github.com/loft-sh/loftctl/v4/pkg/remotecommand"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

const AllWorkspaces = "all"

// RebuildCmd holds the cmd flags
type RebuildCmd struct {
	*flags.GlobalFlags
	Log log.Logger

	Project string
}

// NewRebuildCmd creates a new command
func NewRebuildCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &RebuildCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:   "rebuild",
		Short: "Rebuild a workspace",
		Long: `
#######################################################
################# loft devpod rebuild ##################
#######################################################
	`,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			log.Default.SetFormat(log.TextFormat)

			return cmd.Run(cobraCmd.Context(), args)
		},
	}

	c.Flags().StringVar(&cmd.Project, "project", "", "The project to use")
	_ = c.MarkFlagRequired("project")

	return c
}

func (cmd *RebuildCmd) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("please provide a workspace name")
	}
	targetWorkspace := args[0]

	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	workspace, err := devpodpkg.FindWorkspaceByName(ctx, baseClient, targetWorkspace, cmd.Project)
	if err != nil {
		return err
	}

	opts := struct {
		Recreate bool `json:"recreate"`
	}{Recreate: true}
	rawOpts, err := json.Marshal(opts)
	if err != nil {
		return err
	}
	values := url.Values{"options": []string{string(rawOpts)}, "cliMode": []string{"true"}}
	conn, err := devpodpkg.DialWorkspace(baseClient, workspace, "up", values)
	if err != nil {
		return err
	}

	_, err = remotecommand.ExecuteConn(ctx, conn, os.Stdin, os.Stdout, os.Stderr, cmd.Log.ErrorStreamOnly())
	if err != nil {
		return fmt.Errorf("error executing: %w", err)
	}

	return nil
}
