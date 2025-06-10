package telemetry

import "github.com/loft-sh/vcluster/pkg/config"

type ChartInfo struct {
	Values  *config.VirtualClusterConfig
	Name    string
	Version string
}

type KubernetesVersion struct {
	Major      string `json:"major"`
	Minor      string `json:"minor"`
	GitVersion string `json:"gitVersion"`
}
