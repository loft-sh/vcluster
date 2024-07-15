package constants

const (
	SkipTranslationAnnotation = "vcluster.loft.sh/skip-translate"
	SyncResourceAnnotation    = "vcluster.loft.sh/force-sync"

	PausedAnnotation         = "loft.sh/paused"
	PausedReplicasAnnotation = "loft.sh/paused-replicas"
	PausedDateAnnotation     = "loft.sh/paused-date"

	HostClusterPersistentVolumeAnnotation = "vcluster.loft.sh/host-pv"

	HostClusterVSCAnnotation = "vcluster.loft.sh/host-volumesnapshotcontent"

	// NodeSuffix is the dns suffix for our nodes
	NodeSuffix = "nodes.vcluster.com"

	// KubeletPort is the port we pretend the kubelet is running under
	KubeletPort = int32(10250)
)
