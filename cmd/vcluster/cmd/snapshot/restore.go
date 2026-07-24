package snapshot

import (
	"fmt"
	"os"

	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/snapshot"
	standaloneutil "github.com/loft-sh/vcluster/pkg/util/standalone"
	"github.com/spf13/cobra"
)

var (
	newVCluster bool
)

// isServiceActive reports whether the standalone vcluster systemd unit is
// active on this host. Package-level so tests can stub the probe.
var isServiceActive = standaloneutil.IsServiceActive

func NewRestoreCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore a vCluster",
		Long: `Restores the vCluster backing store from a snapshot. This is an internal
command: it must only run while the vCluster control plane is down, and it is
normally invoked by the vcluster CLI or by the snapshot pod.

On a standalone host, use "vcluster restore --standalone <snapshot-url>" (the
vcluster CLI) instead of calling this command directly: the CLI stops
vcluster.service, runs this command, and restarts the service afterwards.
Direct invocations while vcluster.service is active are refused to protect
the backing store.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// A restore rewrites the backing store and must never run while the
			// standalone control plane is up: it would race the live kine/etcd,
			// and the unit's Restart=always re-spawns a crashed server
			// mid-restore. The vcluster CLI ("vcluster restore --standalone")
			// stops and restarts the service around this command; refuse direct
			// invocations that skipped that envelope. On hosts without systemd
			// (e.g. the in-pod restore path) this reports not-active and the
			// restore proceeds.
			if isServiceActive() {
				return fmt.Errorf(
					"refusing to restore: %s.service is active on this host; use the vcluster CLI (%q), which stops and restarts the service around the restore, or stop it first with %q",
					constants.VClusterStandaloneSystemdServiceName,
					"vcluster restore --standalone <snapshot-url>",
					"systemctl stop "+constants.VClusterStandaloneSystemdServiceName,
				)
			}

			vConfig, err := config.LoadConfig(os.Getenv("VCLUSTER_NAME"))
			if err != nil {
				return err
			}

			envOptions, err := snapshot.ParseOptionsFromEnv()
			if err != nil {
				return fmt.Errorf("failed to parse options from environment: %w", err)
			}
			restoreClient := snapshot.NewRestoreClient(*envOptions, newVCluster)
			return restoreClient.Run(cmd.Context(), vConfig)
		},
	}

	cmd.Flags().BoolVar(&newVCluster, "new-vcluster", false, "Restore a new vCluster from snapshot instead of restoring into an existing vCluster")
	return cmd
}
