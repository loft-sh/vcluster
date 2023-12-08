package registry

type VClusterConfig struct {
	// ChartInfo holds the chart info with which vCluster was deployed
	ChartInfo *ChartInfo `json:"chartInfo,omitempty"`
}

type ChartInfo struct {
	// Name of the chart that was deployed
	Name string
	// Version of the chart that was deployed
	Version string
	// Values of the chart that was deployed
	Values map[string]interface{}
}
