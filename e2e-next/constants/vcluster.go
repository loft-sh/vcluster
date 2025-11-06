package constants

import (
	_ "embed"
)

var (
	//go:embed defaultvcluster.yaml
	DefaultVClusterYAML string
)

const (
	DefaultVClusterLegacyVersion = "0.19.10"
)
