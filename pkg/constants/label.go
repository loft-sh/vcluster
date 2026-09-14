package constants

const (
	VClusterNamespaceLabel = VClusterNamespaceAnnotation
	VClusterNameLabel      = VClusterNameAnnotation

	// SnapshotRequestLabel is used to label ConfigMaps as snapshot requests.
	SnapshotRequestLabel = "vcluster.loft.sh/snapshot-request"

	// RestoreRequestLabel is used to label ConfigMaps as restore requests.
	RestoreRequestLabel = "vcluster.loft.sh/restore-request"
)
