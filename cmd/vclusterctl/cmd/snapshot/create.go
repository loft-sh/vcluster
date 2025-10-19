package snapshot

import (
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/spf13/cobra"
)

type CreateCmd struct {
	*flags.GlobalFlags
	Snapshot       snapshot.Options
	IncludeVolumes bool
	Log            log.Logger
}

func NewCreateCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &CreateCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	_, nameValidator := util.NamedPositionalArgsValidator(true, false, "VCLUSTER_NAME")
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Snapshot a virtual cluster",
		Long: `##############################################################
################# vcluster snapshot create ###################
##############################################################
Snapshot a virtual cluster. The command creates a snapshot
request, which will be processed asynchronously by a vCluster
controller.

Example:
# Snapshot to oci image
vcluster snapshot create my-vcluster oci://ghcr.io/my-user/my-repo:my-tag
# Snapshot to s3 bucket
vcluster snapshot create my-vcluster s3://my-bucket/my-bucket-key
# Snapshot to vCluster container filesystem
vcluster snapshot create my-vcluster container:///data/my-local-snapshot.tar.gz
##############################################################
	`,
		Args:              nameValidator,
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cli.CreateSnapshot(cobraCmd.Context(), args, cmd.GlobalFlags, &cmd.Snapshot, nil, cmd.Log, true)
		},
	}

	// add storage flags
	snapshot.AddFlags(createCmd.Flags(), &cmd.Snapshot)
	return createCmd
}
