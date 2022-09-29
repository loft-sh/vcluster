package pods

const (
	VirtualPathTemplate = "/tmp/vcluster/%s/%s"

	PodLoggingHostpathPath = "/var/log/pods"
	LogHostpathPath        = "/var/log"

	KubeletPodPath = "/var/lib/kubelet/pods"

	PhysicalVolumeNameSuffix = "vcluster-physical"

	PhysicalLogVolumeMountPath     = "/var/vcluster/physical/log/pods"
	PhysicalKubeletVolumeMountPath = "/var/vcluster/physical/kubelet/pods"
)
