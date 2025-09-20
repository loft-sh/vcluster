package snapshot

import (
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/spf13/cobra"
)

type ListRequestsCmd struct {
	*flags.GlobalFlags
	Log log.Logger
}

func NewListRequestsCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	listRequestsCmd := &ListRequestsCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	_, nameValidator := util.NamedPositionalArgsValidator(true, false, "VCLUSTER_NAME")
	cobraCmd := &cobra.Command{
		Use:   "list-requests",
		Short: "List snapshot requests",
		Long: `##############################################################
################# vcluster snapshot list-requests ###################
###################################################################
List all snapshot requests for a virtual cluster.

Example:
vcluster snapshot list-requests my-vcluster
###################################################################
	`,
		Args:              nameValidator,
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cli.ListRequests(cobraCmd.Context(), args, listRequestsCmd.GlobalFlags, listRequestsCmd.Log)
		},
	}

	return cobraCmd
}
