package constants

const (
	VClusterStandaloneEndpointsAnnotation = "vcluster.loft.sh/standalone-endpoints"
	VClusterStandaloneIPAddressEnvVar     = "VCLUSTER_STANDALONE_IP_ADDRESS"
<<<<<<< ours
=======
	VClusterStandaloneDefaultName         = "standalone"

	// Standalone has no host-cluster namespace, so snapshot/restore request
	// ConfigMaps and Secrets live in the virtual cluster's own kube-system.
	VClusterStandaloneSnapshotNamespace = "kube-system"

	// VClusterStandaloneSystemdServiceName is the name of the systemd service name.
	VClusterStandaloneSystemdServiceName = "vcluster"

	// VClusterStandaloneSystemdUnitFile is the systemd unit file created by the standalone installer.
	// Its presence on disk should indicate we are running on a standalone vCluster host.
	VClusterStandaloneSystemdUnitFile = "/etc/systemd/system/" + VClusterStandaloneSystemdServiceName + ".service"

	// VClusterStandaloneSystemdDropInDir holds unit overrides. The installer drops it on
	// --reset-only, the CLI writes the drop-in below into it.
	VClusterStandaloneSystemdDropInDir = VClusterStandaloneSystemdUnitFile + ".d"

	// VClusterStandalonePlatformDropInFile is written by `vcluster platform add standalone`.
	VClusterStandalonePlatformDropInFile = VClusterStandaloneSystemdDropInDir + "/platform.conf"

	// VClusterStandaloneDefaultDataDir is the default standalone data directory used by
	// binary installations on the host.
	VClusterStandaloneDefaultDataDir = "/var/lib/vcluster"

	// VClusterStandaloneConfigDir holds the host's configuration.
	VClusterStandaloneConfigDir = "/etc/vcluster"

	// VClusterStandaloneDefaultConfigPath is the config location for a standalone binary installation.
	// Kept outside the data directory so it survives a data wipe or re-install.
	VClusterStandaloneDefaultConfigPath = VClusterStandaloneConfigDir + "/vcluster.yaml"

	// VClusterStandaloneSecretsDir is root-only, which is what lets the files below be
	// named after the scope they cover rather than the one value they hold today.
	VClusterStandaloneSecretsDir = VClusterStandaloneConfigDir + "/secrets"

	// Secrets the unit loads with EnvironmentFile= rather than Environment=, which systemd
	// serves to any local user over D-Bus.
	VClusterStandalonePlatformEnvFile = VClusterStandaloneSecretsDir + "/platform.env"
	VClusterStandaloneJoinEnvFile     = VClusterStandaloneSecretsDir + "/join.env"

	// StandaloneRuntimeMetadataFileName stores persisted standalone runtime metadata in the data directory.
	StandaloneRuntimeMetadataFileName = "standalone-runtime-metadata"
>>>>>>> theirs
)
