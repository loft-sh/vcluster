package pods

const (
	VIRTUAL_LOGS_PATH_TEMPLATE      = "/tmp/vcluster/%s/%s/log/pods"
	LOGGING_HOSTPATH_PATH           = "/var/log/pods"
	PHYSICAL_LOG_VOLUME_NAME_SUFFIX = "vcluster-physical"
	PHYSICAL_LOG_VOLUME_MOUNT_PATH  = "/var/vcluster/physical/log/pods"
)
