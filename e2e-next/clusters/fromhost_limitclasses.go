package clusters

import _ "embed"

// FromHostLimitClassesVCluster uses label-selector-based fromHost sync for
// ingressClasses, storageClasses, priorityClasses, and runtimeClasses
// (matchLabels: value: one).

//go:embed vcluster-fromhost-limitclasses.yaml
var fromHostLimitClassesVClusterYAML string

var (
	FromHostLimitClassesVClusterName = "fromhost-limitclasses-vcluster"
	FromHostLimitClassesVCluster     = register(FromHostLimitClassesVClusterName, fromHostLimitClassesVClusterYAML)
)
