package constants

const (
	SkipTranslationAnnotation = "vcluster.loft.sh/skip-translate"
	SyncResourceAnnotation    = "vcluster.loft.sh/force-sync"

	PausedReplicasAnnotation = "loft.sh/paused-replicas"
	PausedDateAnnotation     = "loft.sh/paused-date"

	HostClusterPersistentVolumeAnnotation = "vcluster.loft.sh/host-pv"

	HostClusterVSCAnnotation = "vcluster.loft.sh/host-volumesnapshotcontent"

	// NodeSuffix is the dns suffix for our nodes
	NodeSuffix = "nodes.vcluster.com"

	// KubeletPort is the port we pretend the kubelet is running under
	KubeletPort = int32(10250)

	// LoftDirectClusterEndpoint is a cluster annotation that tells the loft cli to use this endpoint instead of
	// the default loft server address to connect to this cluster.
	LoftDirectClusterEndpoint = "loft.sh/direct-cluster-endpoint"

	// LoftDirectClusterEndpointInsecure specifies if we should use insecure connection for this cluster
	LoftDirectClusterEndpointInsecure = "loft.sh/direct-cluster-endpoint-insecure"
)

func PausedAnnotation(isRestore bool) string {
	if isRestore {
		return "loft.sh/restoring"
	}
	return "loft.sh/paused"
}
