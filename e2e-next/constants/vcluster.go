package constants

import (
	_ "embed"
)

var (
	// go:embed vcluster.yaml
	DefaultVClusterYAML string
)

const (
	DefaultVClusterLegacyVersion = "0.19.10"
)
