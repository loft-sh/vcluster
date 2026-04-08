package snapshot

import (
	"cmp"
	"fmt"
	"time"

	"github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/pkg/cli"
	"github.com/loft-sh/vcluster/pkg/cli/completion"
	"github.com/loft-sh/vcluster/pkg/cli/config"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/loft-sh/vcluster/pkg/cli/util"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	"github.com/spf13/cobra"
)

type CreateCmd struct {
	*flags.GlobalFlags
	Snapshot       snapshot.Options
	Driver         string
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
# Snapshot a Docker-based vCluster to a local file
vcluster snapshot create my-vcluster ./my-snapshot.tar.gz --driver docker
# Snapshot with auto-generated filename (my-vcluster-snapshot-<timestamp>.tar.gz)
vcluster snapshot create my-vcluster --driver docker
##############################################################
	`,
		Args:              nameValidator,
		ValidArgsFunction: completion.NewValidVClusterNameFunc(globalFlags),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			cfg := cmd.LoadedConfig(cmd.Log)
			driverType, err := config.ParseDriverType(cmp.Or(cmd.Driver, string(cfg.Driver.Type)))
			if err != nil {
				return fmt.Errorf("parse driver type: %w", err)
			}
			if driverType == config.DockerDriver {
				vClusterName := args[0]
				outputPath := ""
				if len(args) >= 2 {
					outputPath = args[1]
				} else {
					outputPath = fmt.Sprintf("%s-snapshot-%s.tar.gz", vClusterName, time.Now().Format("2006-01-02T15-04-05"))
				}
				return cli.SnapshotDocker(cobraCmd.Context(), cmd.GlobalFlags, vClusterName, outputPath, cmd.Log)
			}
			return cli.CreateSnapshot(cobraCmd.Context(), args, cmd.GlobalFlags, &cmd.Snapshot, nil, cmd.Log, true)
		},
	}

	createCmd.Flags().StringVar(&cmd.Driver, "driver", "", "The driver to use for managing the virtual cluster, can be either helm, platform, or docker.")
	// add storage flags
	snapshot.AddFlags(createCmd.Flags(), &cmd.Snapshot)
	return createCmd
}
