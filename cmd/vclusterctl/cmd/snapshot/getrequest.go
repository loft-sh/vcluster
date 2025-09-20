package snapshot

import (
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/spf13/cobra"
)

type GetRequestCmd struct {
	*flags.GlobalFlags
	Log log.Logger
}

func NewGetRequestCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &GetRequestCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	_, nameValidator := util.NamedPositionalArgsValidator(true, false, "VCLUSTER_NAME")
	createCmd := &cobra.Command{
		Use:   "get-request",
		Short: "Get a snapshot request",
		Long: `##############################################################
################# vcluster snapshot get-request ###################
###################################################################
Get a snapshot request.

Example:
vcluster snapshot get-request my-vcluster snapshot-request-hy91d
###################################################################
	`,
		Args:              nameValidator,
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cli.GetRequest(cobraCmd.Context(), args, cmd.GlobalFlags, cmd.Log)
		},
	}

	return createCmd
}
