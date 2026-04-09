package constants

const (
	VClusterStandaloneEndpointsAnnotation = "vcluster.loft.sh/standalone-endpoints"
	VClusterStandaloneEnvVar              = "VCLUSTER_STANDALONE"
	VClusterStandaloneIPAddressEnvVar     = "VCLUSTER_STANDALONE_IP_ADDRESS"

	// Standalone has no host-cluster namespace, so snapshot/restore request
	// ConfigMaps and Secrets live in the virtual cluster's own kube-system.
	StandaloneSnapshotNamespace = "kube-system"

	// VClusterServiceFile is the systemd unit file created by the standalone installer.
	// Its presence on disk should indicate we are running on a standalone vCluster host.
	VClusterServiceFile = "/etc/systemd/system/vcluster.service"
)
