package snapshot

import (
	"fmt"

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

	Snapshot   snapshot.Options
	Pod        pod.Options
	Standalone bool

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
		Args: func(cobraCmd *cobra.Command, args []string) error {
			if rootCmd.Standalone {
				if len(args) != 1 {
					return fmt.Errorf("%s\nInvalid Args: received %d arguments, expected 1, please specify: %q\nRun with --help for more details on arguments", cobraCmd.UseLine(), len(args), "SNAPSHOT_URL")
				}
				return nil
			}
			return nameValidator(cobraCmd, args)
		},
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cli.CreateSnapshot(cobraCmd.Context(), args, rootCmd.GlobalFlags, &rootCmd.Snapshot, &rootCmd.Pod, rootCmd.Log, false, rootCmd.Standalone)
		},
	}

	// add storage flags
	pod.AddFlags(cobraCmd.Flags(), &rootCmd.Pod, false)
	cobraCmd.Flags().BoolVar(&rootCmd.Standalone, "standalone", false, "Target the local standalone vCluster on this host")
	snapshot.AddFlags(cobraCmd.Flags(), &rootCmd.Snapshot)

	// add subcommands
	cobraCmd.AddCommand(NewCreateCmd(globalFlags))
	cobraCmd.AddCommand(NewGetCmd(globalFlags))

	return cobraCmd
}
