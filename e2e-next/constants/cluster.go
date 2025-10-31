package constants

var vclusterCluster string

func GetClusterName() string {
	if vclusterCluster == "" {
		return "vcluster"
	}
	return vclusterCluster
}

func SetClusterName(name string) {
	vclusterCluster = name
}
