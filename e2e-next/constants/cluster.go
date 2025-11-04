package constants

var vclusterCluster string

func GetClusterName() string {
	if vclusterCluster == "" {
		return "kind-cluster"
	}
	return vclusterCluster
}

func SetClusterName(name string) {
	vclusterCluster = name
}
