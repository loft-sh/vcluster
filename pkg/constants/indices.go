package constants

import "time"

const (
	IndexByPhysicalName  = "IndexByPhysicalName"
	IndexByVirtualName   = "IndexByVirtualName"
	IndexByAssigned      = "IndexByAssigned"
	IndexByStorageClass  = "IndexByStorageClass"
	IndexByIngressSecret = "IndexByIngressSecret"
	IndexByPodSecret     = "IndexByPodSecret"
	IndexByConfigMap     = "IndexByConfigMap"
	// IndexByHostName is used to map rewritten hostnames(advertised as node addresses) to nodenames
	IndexByHostName = "IndexByHostName"

	IndexByClusterIP = "IndexByClusterIP"

	// IndexRunningNonVClusterPodsByNode is only used when the vcluster scheduler is enabled.
	// It maps non-vcluster pods on the node to the node name, so that the node syncer may
	// calculate the allocatable resources on the node.
	IndexRunningNonVClusterPodsByNode = "IndexRunningNonVClusterPodsByNode"
)

const DefaultCacheSyncTimeout = time.Minute * 15
