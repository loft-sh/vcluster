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

type RestoreCmd struct {
	*flags.GlobalFlags

	Snapshot       snapshot.Options
	Pod            pod.Options
	RestoreVolumes bool

	Log log.Logger
}

// NewRestore creates a new command
func NewRestore(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &RestoreCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	useLine, nameValidator := util.NamedPositionalArgsValidator(true, false, "VCLUSTER_NAME")
	cobraCmd := &cobra.Command{
		Use:   "restore" + useLine,
		Short: "Restores a virtual cluster from snapshot",
		Long: `#######################################################
################# vcluster restore ####################
#######################################################
Restore a virtual cluster.

Example:
# Restore from oci image
vcluster restore my-vcluster oci://ghcr.io/my-user/my-repo:my-tag
# Restore from s3 bucket
vcluster restore my-vcluster s3://my-bucket/my-bucket-key
# Restore from vCluster container filesystem
vcluster restore my-vcluster container:///data/my-local-snapshot.tar.gz
#######################################################
	`,
		Args:              nameValidator,
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cli.Restore(cobraCmd.Context(), args, cmd.GlobalFlags, &cmd.Snapshot, &cmd.Pod, false, cmd.RestoreVolumes, cmd.Log)
		},
	}

	// add storage flags
	pod.AddFlags(cobraCmd.Flags(), &cmd.Pod, true)
	cobraCmd.Flags().BoolVar(&cmd.RestoreVolumes, "restore-volumes", false, "Restore volumes from volume snapshots")
	return cobraCmd
}
