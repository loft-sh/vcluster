package constants

import "os"

var hostCluster string

func GetHostClusterName() string {
	if hostCluster == "" {
		if kindName, ok := os.LookupEnv("KIND_NAME"); ok && kindName != "" {
			return kindName
		}

		return "kind-cluster"
	}
	return hostCluster
}

func SetHostClusterName(name string) {
	hostCluster = name
}
