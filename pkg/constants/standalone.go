package constants

const (
	VClusterStandaloneEndpointsAnnotation = "vcluster.loft.sh/standalone-endpoints"
	VClusterStandaloneIPAddressEnvVar     = "VCLUSTER_STANDALONE_IP_ADDRESS"

	// Set on default/kubernetes Service by the syncer so the CLI can detect
	// standalone vClusters without any host-cluster context.
	VClusterStandaloneNameAnnotation    = "vcluster.loft.sh/standalone-name"
	VClusterStandaloneVersionAnnotation = "vcluster.loft.sh/standalone-version"

	// Standalone has no host-cluster namespace, so snapshot/restore request
	// ConfigMaps and Secrets live in the virtual cluster's own kube-system.
	StandaloneSnapshotNamespace = "kube-system"
)
