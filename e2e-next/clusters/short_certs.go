package clusters

import _ "embed"

// ShortCertsVCluster uses DEVELOPMENT=true and VCLUSTER_CERTS_VALIDITYPERIOD=3m
// to create very short-lived serving certificates for testing runtime cert rotation.

//go:embed vcluster-short-certs.yaml
var shortCertsVClusterYAML string

var (
	ShortCertsVClusterName = "short-certs-vcluster"
	ShortCertsVCluster     = register(ShortCertsVClusterName, shortCertsVClusterYAML)
)
