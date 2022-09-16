package pods

const (
	VirtualLogsPathTemplate     = "/tmp/vcluster/%s/%s/log"
	PodLoggingHostpathPath      = "/var/log/pods"
	LogHostpathPath             = "/var/log"
	PhysicalLogVolumeNameSuffix = "vcluster-physical"
	PhysicalLogVolumeMountPath  = "/var/vcluster/physical/log/pods"
)
