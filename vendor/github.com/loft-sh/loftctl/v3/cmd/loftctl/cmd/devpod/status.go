package devpod

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	storagev1 "github.com/loft-sh/api/v3/pkg/apis/storage/v1"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/remotecommand"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// StatusCmd holds the cmd flags
type StatusCmd struct {
	*flags.GlobalFlags

	Log log.Logger
}

// NewStatusCmd creates a new command
func NewStatusCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &StatusCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:   "status",
		Short: "Runs status on a workspace",
		Long: `
#######################################################
################# loft devpod status ##################
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), os.Stdin, os.Stdout, os.Stderr)
		},
	}

	return c
}

func (cmd *StatusCmd) Run(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	workspace, err := findWorkspace(ctx, baseClient)
	if err != nil {
		return err
	} else if workspace == nil {
		out, err := json.Marshal(&storagev1.WorkspaceStatusResult{
			ID:       os.Getenv(LOFT_WORKSPACE_ID),
			Context:  os.Getenv(LOFT_WORKSPACE_CONTEXT),
			State:    string(storagev1.WorkspaceStatusNotFound),
			Provider: os.Getenv(LOFT_WORKSPACE_PROVIDER),
		})
		if err != nil {
			return err
		}

		fmt.Println(string(out))
		return nil
	}

	conn, err := dialWorkspace(baseClient, workspace, "getstatus", optionsFromEnv(storagev1.DevPodFlagsStatus))
	if err != nil {
		return err
	}

	_, err = remotecommand.ExecuteConn(ctx, conn, stdin, stdout, stderr, cmd.Log.ErrorStreamOnly())
	if err != nil {
		return fmt.Errorf("error executing: %w", err)
	}

	return nil
}
