package clusters

import _ "embed"

// ServiceSyncVCluster has specific replicateServices mappings for service
// replication tests (toHost/fromHost with named service mappings).

//go:embed vcluster-servicesync.yaml
var serviceSyncVClusterYAML string

var (
	ServiceSyncVClusterName = "service-sync-vcluster"
	ServiceSyncVCluster     = register(ServiceSyncVClusterName, serviceSyncVClusterYAML)
)
