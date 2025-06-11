package cmd

import (
	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/loft-sh/vcluster/pkg/snapshot/pod"
	"github.com/spf13/cobra"
)

type SnapshotCmd struct {
	*flags.GlobalFlags

	Snapshot snapshot.Options
	Pod      pod.Options

	Log log.Logger
}

// NewSnapshot creates a new command
func NewSnapshot(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &SnapshotCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	useLine, nameValidator := util.NamedPositionalArgsValidator(true, false, "VCLUSTER_NAME")
	cobraCmd := &cobra.Command{
		Use:   "snapshot" + useLine,
		Short: "Snapshot a virtual cluster",
		Long: `#######################################################
################# vcluster snapshot ###################
#######################################################
Snapshot a virtual cluster.

Example:
# Snapshot to oci image
vcluster snapshot my-vcluster oci://ghcr.io/my-user/my-repo:my-tag
# Snapshot to s3 bucket
vcluster snapshot my-vcluster s3://my-bucket/my-bucket-key
# Snapshot to vCluster container filesystem
vcluster snapshot my-vcluster container:///data/my-local-snapshot.tar.gz
#######################################################
	`,
		Args:              nameValidator,
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cli.Snapshot(cobraCmd.Context(), args, cmd.GlobalFlags, &cmd.Snapshot, &cmd.Pod, cmd.Log)
		},
	}

	// add storage flags
	pod.AddFlags(cobraCmd.Flags(), &cmd.Pod, false)
	snapshot.AddFlags(cobraCmd.Flags(), &cmd.Snapshot)
	return cobraCmd
}
