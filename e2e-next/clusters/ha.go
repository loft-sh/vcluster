package clusters

import _ "embed"

// HAVCluster runs with 3 replicas (statefulset + etcd + coredns) for
// high-availability cert rotation tests.

//go:embed vcluster-ha.yaml
var haVClusterYAML string

var (
	HAVClusterName = "ha-certs-vcluster"
	HAVCluster     = register(HAVClusterName, haVClusterYAML)
)
