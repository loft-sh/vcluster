package clusters

import _ "embed"

// CommonVCluster is the comprehensive vCluster used by the main e2e suite.
// Most tests run against this cluster.

//go:embed vcluster-default.yaml
var defaultVClusterYAML string

var (
	CommonVClusterName = "common-vcluster"
	CommonVCluster     = register(CommonVClusterName, defaultVClusterYAML)
)

// Aliases for backward compatibility - all point to CommonVCluster.
var (
	K8sDefaultEndpointVCluster     = CommonVCluster
	K8sDefaultEndpointVClusterName = CommonVClusterName
	NodesVCluster                  = CommonVCluster
	NodesVClusterName              = CommonVClusterName
	HelmChartsVCluster             = CommonVCluster
	HelmChartsVClusterName         = CommonVClusterName
	InitManifestsVCluster          = CommonVCluster
	InitManifestsVClusterName      = CommonVClusterName
	FromHostConfigMapsVCluster     = CommonVCluster
	FromHostConfigMapsVClusterName = CommonVClusterName
	FromHostSecretsVCluster        = CommonVCluster
	FromHostSecretsVClusterName    = CommonVClusterName
)
