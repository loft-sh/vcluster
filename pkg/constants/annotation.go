package constants

const (
	SkipTranslationAnnotation = "vcluster.loft.sh/skip-translate"
	SyncResourceAnnotation    = "vcluster.loft.sh/force-sync"

	PausedReplicasAnnotation = "loft.sh/paused-replicas"
	PausedDateAnnotation     = "loft.sh/paused-date"

	HostClusterPersistentVolumeAnnotation = "vcluster.loft.sh/host-pv"

	HostClusterVSCAnnotation = "vcluster.loft.sh/host-volumesnapshotcontent"

	// PreserveRequestMirrorFiltersAnnotation when set to "true" on a host HTTPRoute, tells
	// vCluster's HTTPRoute syncer to keep any RequestMirror filters that an external host-side
	// controller has appended to the route's rules, instead of stripping them when reconciling
	// the spec from virtual. This is the extension hook for host controllers that augment routes
	// post-sync (e.g. for activity tracking or traffic mirroring);
	PreserveRequestMirrorFiltersAnnotation = "vcluster.loft.sh/preserve-request-mirror-filters"

	// PreserveHostRuleAnnotation when set on a host HTTPRoute names a single
	// HTTPRouteRule (by HTTPRouteRule.Name) whose host-side definition vCluster's
	// HTTPRoute syncer must preserve verbatim onto the desired host spec during
	// virtual->host reconciliation. The named rule is re-prepended to the desired
	// spec at index 0 so it survives spec re-derivation from the virtual route.
	// The annotation is the extension hook for external host-side controllers that
	// need to inject a high-priority managed rule that is not visible to the tenant.
	PreserveHostRuleAnnotation = "vcluster.loft.sh/preserve-host-rule"

	// NodeSuffix is the dns suffix for our nodes
	NodeSuffix = "nodes.vcluster.com"

	// KubeletPort is the port we pretend the kubelet is running under
	KubeletPort = int32(10250)

	// LoftDirectClusterEndpoint is a cluster annotation that tells the loft cli to use this endpoint instead of
	// the default loft server address to connect to this cluster.
	LoftDirectClusterEndpoint = "loft.sh/direct-cluster-endpoint"

	// LoftDirectClusterEndpointInsecure specifies if we should use insecure connection for this cluster
	LoftDirectClusterEndpointInsecure = "loft.sh/direct-cluster-endpoint-insecure"

	VClusterNamespaceAnnotation = "vcluster.loft.sh/vcluster-namespace"
	VClusterNameAnnotation      = "vcluster.loft.sh/vcluster-name"
)

func PausedAnnotation(isRestore bool) string {
	if isRestore {
		return "loft.sh/restoring"
	}
	return "loft.sh/paused"
}
