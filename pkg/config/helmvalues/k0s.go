package helmvalues

type K0s struct {
	BaseHelm
	AutoDeletePersistentVolumeClaims bool           `json:"autoDeletePersistentVolumeClaims,omitempty"`
	VCluster                         VClusterValues `json:"vcluster,omitempty"`
	Syncer                           SyncerValues   `json:"syncer,omitempty"`
}
