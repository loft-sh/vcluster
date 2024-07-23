package constants

const (
	K8sKineEndpoint = "unix:///data/kine.sock"
	K3sKineEndpoint = "unix:///data/server/kine.sock"
	K0sKineEndpoint = "unix:///run/k0s/kine/kine.sock:2379"

	// DefaultVClusterConfigLocation is the default location of the vCluster config within the container
	DefaultVClusterConfigLocation = "/var/vcluster/config.yaml"
)
