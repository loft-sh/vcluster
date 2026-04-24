package cmd

import (
	"cmp"
	"fmt"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	snapshotazure "github.com/loft-sh/vcluster/pkg/snapshot/azure"
	"github.com/loft-sh/vcluster/pkg/snapshot/pod"
	"github.com/spf13/cobra"
)

type RestoreCmd struct {
	*flags.GlobalFlags

	Snapshot       snapshot.Options
	Pod            pod.Options
	Driver         string
	Name           string
	RestoreVolumes bool
	Standalone     bool

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
# Restore a Docker-based vCluster from a local snapshot file
vcluster restore my-vcluster ./my-snapshot.tar.gz --driver docker
# Restore with a different name
vcluster restore my-new-name ./my-snapshot.tar.gz --driver docker
#######################################################
	`,
		Args: func(cobraCmd *cobra.Command, args []string) error {
			if cmd.Standalone {
				if len(args) != 1 {
					return fmt.Errorf("%s\nInvalid Args: received %d arguments, expected 1, please specify: %q\nRun with --help for more details on arguments", cobraCmd.UseLine(), len(args), "SNAPSHOT_URL")
				}
				return nil
			}
			return nameValidator(cobraCmd, args)
		},
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			cfg := cmd.LoadedConfig(cmd.Log)
			driverType, err := config.ParseDriverType(cmp.Or(cmd.Driver, string(cfg.Driver.Type)))
			if err != nil {
				return fmt.Errorf("parse driver type: %w", err)
			}
			if cmd.Standalone && driverType == config.DockerDriver {
				return fmt.Errorf("--standalone cannot be used with --driver docker")
			}
			if driverType == config.DockerDriver {
				if len(args) < 2 {
					return fmt.Errorf("usage: vcluster restore VCLUSTER_NAME SNAPSHOT_FILE --driver docker")
				}
				return cli.RestoreDocker(cobraCmd.Context(), cmd.GlobalFlags, args[1], args[0], nil, cmd.Log)
			}
			return cli.Restore(cobraCmd.Context(), args, cmd.GlobalFlags, &cmd.Snapshot, &cmd.Pod, false, cmd.RestoreVolumes, cmd.Standalone, cmd.Log)
		},
	}

	cobraCmd.Flags().StringVar(&cmd.Driver, "driver", "", "The driver to use for managing the virtual cluster, can be either helm, platform, or docker.")
	// add storage flags
	pod.AddFlags(cobraCmd.Flags(), &cmd.Pod, true)
	cobraCmd.Flags().BoolVar(&cmd.RestoreVolumes, "restore-volumes", false, "Restore volumes from volume snapshots")
	cobraCmd.Flags().BoolVar(&cmd.Standalone, "standalone", false, "Target the local standalone vCluster on this host")
	snapshotazure.AddFlags(cobraCmd.Flags(), &cmd.Snapshot.Azure)
	return cobraCmd
}
