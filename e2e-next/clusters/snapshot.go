package clusters

import (
	_ "embed"

	setupcsi "github.com/loft-sh/vcluster/e2e-next/setup"
)

// SnapshotVCluster has PVC/PV sync + snapshot-data volume mount.
// PreSetup installs the CSI hostpath driver and creates the snapshot-data PVC
// in the host namespace before the vCluster is provisioned.

//go:embed vcluster-snapshot.yaml
var snapshotVClusterYAML string

var (
	SnapshotVClusterName = "snapshot-vcluster"
	SnapshotVCluster     = registerWith(SnapshotVClusterName, snapshotVClusterYAML,
		[]RegisterOption{WithPreSetup(setupcsi.SnapshotPreSetup(SnapshotVClusterName))},
	)
)
