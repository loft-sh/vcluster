package clusters

import _ "embed"

// HAShortCertsVCluster runs with 2 replicas and short-lived certificates
// (DEVELOPMENT=true, 3m cert validity, 15s check interval) for testing
// HA-coordinated cert rotation via the rotation lease.

//go:embed vcluster-ha-short-certs.yaml
var haShortCertsVClusterYAML string

var (
	HAShortCertsVClusterName = "ha-short-certs-vcluster"
	HAShortCertsVCluster     = register(HAShortCertsVClusterName, haShortCertsVClusterYAML)
)
