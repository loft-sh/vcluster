package registry

const (
	// ArtifactType is the vCluster artifact type
	ArtifactType = "application/vnd.loft.vcluster"

	// ConfigMediaType is the reserved media type for the vCluster config
	ConfigMediaType = "application/vnd.loft.vcluster.config.v1+json"

	// EtcdLayerMediaType is the reserved media type for the etcd snapshot
	EtcdLayerMediaType = "application/vnd.loft.vcluster.etcd.v1.tar+gzip"

	// PersistentVolumeLayerMediaType is the reserved media type for persistent volumes
	PersistentVolumeLayerMediaType = "application/vnd.loft.vcluster.pv.v1.tar+gzip"
)
