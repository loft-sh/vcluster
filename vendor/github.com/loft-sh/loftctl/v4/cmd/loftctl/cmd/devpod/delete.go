package devpod

import (
	"context"
	"fmt"
	"io"
	"os"

	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
	"github.com/loft-sh/loftctl/v4/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v4/pkg/client"
	devpodpkg "github.com/loft-sh/loftctl/v4/pkg/devpod"
	"github.com/loft-sh/loftctl/v4/pkg/remotecommand"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// DeleteCmd holds the cmd flags
type DeleteCmd struct {
	*flags.GlobalFlags

	Log log.Logger
}

// NewDeleteCmd creates a new command
func NewDeleteCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &DeleteCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Hidden: true,
		Use:    "delete",
		Short:  "Runs delete on a workspace",
		Long: `
#######################################################
################# loft devpod delete ##################
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), os.Stdin, os.Stdout, os.Stderr)
		},
	}

	return c
}

func (cmd *DeleteCmd) Run(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	baseClient, err := client.InitClientFromPath(ctx, cmd.Config)
	if err != nil {
		return err
	}

	info, err := devpodpkg.GetWorkspaceInfoFromEnv()
	if err != nil {
		return err
	}
	workspace, err := devpodpkg.FindWorkspace(ctx, baseClient, info.UID, info.ProjectName)
	if err != nil {
		return err
	} else if workspace == nil {
		return fmt.Errorf("couldn't find workspace")
	}

	conn, err := devpodpkg.DialWorkspace(baseClient, workspace, "delete", devpodpkg.OptionsFromEnv(storagev1.DevPodFlagsDelete))
	if err != nil {
		return err
	}

	_, err = remotecommand.ExecuteConn(ctx, conn, stdin, stdout, stderr, cmd.Log.ErrorStreamOnly())
	if err != nil {
		return fmt.Errorf("error executing: %w", err)
	}

	return nil
}
