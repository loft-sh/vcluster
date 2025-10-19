package snapshot

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

type RootCmd struct {
	*flags.GlobalFlags

	Snapshot snapshot.Options
	Pod      pod.Options

	Log log.Logger
}

// NewSnapshot creates a new command
func NewSnapshot(globalFlags *flags.GlobalFlags) *cobra.Command {
	rootCmd := &RootCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	useLine, nameValidator := util.NamedPositionalArgsValidator(true, false, "VCLUSTER_NAME")
	cobraCmd := &cobra.Command{
		Use:   "snapshot" + useLine,
		Short: "Snapshot a virtual cluster (deprecated, use 'vcluster snapshot create' instead)",
		Long: `#######################################################
################# vcluster snapshot ###################
#######################################################
Snapshot a virtual cluster. Deprecated, use 'vcluster snapshot create' instead.

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
			return cli.CreateSnapshot(cobraCmd.Context(), args, rootCmd.GlobalFlags, &rootCmd.Snapshot, &rootCmd.Pod, rootCmd.Log, false)
		},
	}

	// add storage flags
	pod.AddFlags(cobraCmd.Flags(), &rootCmd.Pod, false)
	snapshot.AddFlags(cobraCmd.Flags(), &rootCmd.Snapshot)

	// add subcommands
	cobraCmd.AddCommand(NewCreateCmd(globalFlags))
	cobraCmd.AddCommand(NewGetCmd(globalFlags))

	return cobraCmd
}
