package clusters

import _ "embed"

// CertsVCluster is a dedicated single-replica vCluster for cert rotation tests.
// It uses deploy etcd (not embedded) so cert secrets exist to rotate.
// This cluster is separate from CommonVCluster because cert rotation restarts the
// vcluster pod, which would kill the proxy for any parallel tests sharing the cluster.

//go:embed vcluster-certs.yaml
var certsVClusterYAML string

var (
	CertsVClusterName = "certs-vcluster"
	CertsVCluster     = register(CertsVClusterName, certsVClusterYAML)
)
