package devpod

import (
	"context"
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

// SshCmd holds the cmd flags
type SshCmd struct {
	*flags.GlobalFlags

	Log log.Logger
}

// NewSshCmd creates a new command
func NewSshCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &SshCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:   "ssh",
		Short: "Runs ssh on a workspace",
		Long: `
#######################################################
################### loft devpod ssh ###################
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context(), os.Stdin, os.Stdout, os.Stderr)
		},
	}

	return c
}

func (cmd *SshCmd) Run(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	baseClient, err := client.NewClientFromPath(cmd.Config)
	if err != nil {
		return err
	}

	workspace, err := findWorkspace(ctx, baseClient)
	if err != nil {
		return err
	} else if workspace == nil {
		return fmt.Errorf("couldn't find workspace")
	}

	conn, err := dialWorkspace(baseClient, workspace, "ssh", optionsFromEnv(storagev1.DevPodFlagsSsh))
	if err != nil {
		return err
	}

	_, err = remotecommand.ExecuteConn(ctx, conn, stdin, stdout, stderr, cmd.Log.ErrorStreamOnly())
	if err != nil {
		return fmt.Errorf("error executing: %w", err)
	}

	return nil
}
