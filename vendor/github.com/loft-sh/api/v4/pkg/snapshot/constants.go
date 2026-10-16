package snapshot

import "time"

const (
	// APIVersion is the snapshot request API version.
	APIVersion = "v1beta1"

	// RequestKey is the ConfigMap data key that stores a snapshot request.
	RequestKey = "snapshotRequest"

	// OptionsKey is the Secret data key that stores snapshot options.
	OptionsKey = "snapshotOptions"

	// SnapshotRequestLabel marks ConfigMaps and Secrets as snapshot request resources.
	SnapshotRequestLabel = "vcluster.loft.sh/snapshot-request"

	// RestoreRequestLabel marks ConfigMaps and Secrets as restore request resources.
	RestoreRequestLabel = "vcluster.loft.sh/restore-request"

	// VClusterNamespaceLabel stores the namespace of the vCluster that owns the request.
	VClusterNamespaceLabel = "vcluster.loft.sh/vcluster-namespace"

	// VClusterNameLabel stores the name of the vCluster that owns the request.
	VClusterNameLabel = "vcluster.loft.sh/vcluster-name"

	// SnapshotReleaseKey stores info about the vCluster Helm release in the snapshot archive.
	SnapshotReleaseKey = "/vcluster/snapshot/release"

	DefaultRequestTTL = 24 * time.Hour
)
