package pods

const (
	VirtualLogsPathTemplate     = "/tmp/vcluster/%s/%s/log/pods"
	LoggingHostpathPath         = "/var/log/pods"
	PhysicalLogVolumeNameSuffix = "vcluster-physical"
	PhysicalLogVolumeMountPath  = "/var/vcluster/physical/log/pods"
)
