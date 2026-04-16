package constants

const (
	VClusterStandaloneEndpointsAnnotation = "vcluster.loft.sh/standalone-endpoints"
	VClusterStandalonePortAnnotation      = "vcluster.loft.sh/standalone-port"
	VClusterStandaloneEnvVar              = "VCLUSTER_STANDALONE"
	VClusterStandaloneIPAddressEnvVar     = "VCLUSTER_STANDALONE_IP_ADDRESS"
	VClusterStandaloneDefaultName         = "standalone"

	// VClusterStandaloneCLISelector is a CLI-only alias that tells commands to target
	// the local standalone service on this host. It triggers local systemd discovery
	// and uses the standalone kubeconfig from disk; it is not the standalone runtime
	// vCluster name used in request resources.
	VClusterStandaloneCLISelector = "local-standalone"

	// Standalone has no host-cluster namespace, so snapshot/restore request
	// ConfigMaps and Secrets live in the virtual cluster's own kube-system.
	VClusterStandaloneSnapshotNamespace = "kube-system"

	// VClusterStandaloneSystemdServiceName is the name of the systemd service name.
	VClusterStandaloneSystemdServiceName = "vcluster"

	// VClusterStandaloneSystemdUnitFile is the systemd unit file created by the standalone installer.
	// Its presence on disk should indicate we are running on a standalone vCluster host.
	VClusterStandaloneSystemdUnitFile = "/etc/systemd/system/" + VClusterStandaloneSystemdServiceName + ".service"

	// VClusterStandaloneDefaultDataDir is the default standalone data directory used by
	// binary installations on the host.
	VClusterStandaloneDefaultDataDir = "/var/lib/vcluster"

	// VClusterStandaloneDefaultConfigPath is the config location for a standalone binary installation.
	// Kept outside the data directory so it survives a data wipe or re-install.
	VClusterStandaloneDefaultConfigPath = "/etc/vcluster/vcluster.yaml"
)
