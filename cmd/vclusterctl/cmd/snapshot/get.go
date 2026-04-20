package snapshot

import (
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/spf13/cobra"
)

type GetCmd struct {
	*flags.GlobalFlags
	Snapshot   snapshot.Options
	Standalone bool
	Log        log.Logger
}

func NewGetCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &GetCmd{
		GlobalFlags: globalFlags,
		Log:         log.GetInstance(),
	}

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get virtual cluster snapshot",
		Long: `##############################################################
################### vcluster snapshot get ####################
##############################################################
Get virtual cluster snapshot.

Example:
# Get snapshot from oci image
vcluster snapshot get my-vcluster oci://ghcr.io/my-user/my-repo:my-tag
# Get snapshot from s3 bucket
vcluster snapshot get my-vcluster s3://my-bucket/my-bucket-key
# Get snapshot from vCluster container filesystem
vcluster snapshot get my-vcluster container:///data/my-local-snapshot.tar.gz
##############################################################
	`,
		Args: func(cobraCmd *cobra.Command, args []string) error {
			if cmd.Standalone {
				if len(args) != 1 {
					return fmt.Errorf("%s\nInvalid Args: received %d arguments, expected 1, please specify: %q\nRun with --help for more details on arguments", cobraCmd.UseLine(), len(args), "SNAPSHOT_URL")
				}
				return nil
			}
			_, nameValidator := util.NamedPositionalArgsValidator(true, false, "VCLUSTER_NAME")
			return nameValidator(cobraCmd, args)
		},
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cli.GetSnapshots(cobraCmd.Context(), args, cmd.GlobalFlags, &cmd.Snapshot, cmd.Log, cmd.Standalone)
		},
	}

	// add storage flags
	getCmd.Flags().BoolVar(&cmd.Standalone, "standalone", false, "Target the local standalone vCluster on this host")
	snapshot.AddFlags(getCmd.Flags(), &cmd.Snapshot)
	return getCmd
}
