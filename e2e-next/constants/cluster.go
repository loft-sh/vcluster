package constants

var hostCluster string

func GetHostClusterName() string {
	if hostCluster == "" {
		return "kind-cluster"
	}
	return hostCluster
}

func SetHostClusterName(name string) {
	hostCluster = name
}
