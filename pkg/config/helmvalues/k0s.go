package helmvalues

type K0s struct {
	BaseHelm
	VCluster VClusterValues `json:"vcluster,omitempty"`
	Syncer   SyncerValues   `json:"syncer,omitempty"`
}
