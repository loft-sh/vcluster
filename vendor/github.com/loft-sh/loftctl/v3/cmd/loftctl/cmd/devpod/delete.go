package devpod

import (
	"context"
	"fmt"
	"os"

	storagev1 "github.com/loft-sh/api/v3/pkg/apis/storage/v1"
	"github.com/loft-sh/loftctl/v3/cmd/loftctl/flags"
	"github.com/loft-sh/loftctl/v3/pkg/client"
	"github.com/loft-sh/loftctl/v3/pkg/remotecommand"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

// DeleteCmd holds the cmd flags
type DeleteCmd struct {
	*flags.GlobalFlags

	log log.Logger
}

// NewDeleteCmd creates a new command
func NewDeleteCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &DeleteCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:   "delete",
		Short: "Runs delete on a workspace",
		Long: `
#######################################################
################# loft devpod delete ##################
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	return c
}

func (cmd *DeleteCmd) Run(ctx context.Context) error {
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

	conn, err := dialWorkspace(baseClient, workspace, "delete", optionsFromEnv(storagev1.DevPodFlagsDelete))
	if err != nil {
		return err
	}

	_, err = remotecommand.ExecuteConn(ctx, conn, os.Stdin, os.Stdout, os.Stderr, cmd.log.ErrorStreamOnly())
	if err != nil {
		return fmt.Errorf("error executing: %w", err)
	}

	return nil
}
