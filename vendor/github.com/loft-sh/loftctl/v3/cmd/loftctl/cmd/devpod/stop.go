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

var (
	LOFT_WORKSPACE_ID       = "WORKSPACE_ID"
	LOFT_WORKSPACE_CONTEXT  = "WORKSPACE_CONTEXT"
	LOFT_WORKSPACE_PROVIDER = "WORKSPACE_PROVIDER"

	LOFT_WORKSPACE_UID = "WORKSPACE_UID"

	LOFT_PROJECT_OPTION = "LOFT_PROJECT"

	LOFT_TEMPLATE_OPTION         = "LOFT_TEMPLATE"
	LOFT_TEMPLATE_VERSION_OPTION = "LOFT_TEMPLATE_VERSION"
)

// StopCmd holds the cmd flags
type StopCmd struct {
	*flags.GlobalFlags

	log log.Logger
}

// NewStopCmd creates a new command
func NewStopCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &StopCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:   "stop",
		Short: "Runs stop on a workspace",
		Long: `
#######################################################
################## loft devpod stop ###################
#######################################################
	`,
		Args: cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	return c
}

func (cmd *StopCmd) Run(ctx context.Context) error {
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

	conn, err := dialWorkspace(baseClient, workspace, "stop", optionsFromEnv(storagev1.DevPodFlagsStop))
	if err != nil {
		return err
	}

	_, err = remotecommand.ExecuteConn(ctx, conn, os.Stdin, os.Stdout, os.Stderr, cmd.log.ErrorStreamOnly())
	if err != nil {
		return fmt.Errorf("error executing: %w", err)
	}

	return nil
}
